package nacos

import (
	_ "context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

type ServiceDiscovery struct {
	client         naming_client.INamingClient
	services       map[string][]ServiceInstance
	mu             sync.RWMutex
	stopChan       chan struct{}
	connected      bool
	connectChan    chan struct{}
	subscriptions  map[string]struct{}
	connectionOnce sync.Once
}

type ServiceInstance struct {
	InstanceId  string            `json:"instanceId"`
	ServiceName string            `json:"serviceName"`
	IP          string            `json:"ip"`
	Port        uint64            `json:"port"`
	Weight      float64           `json:"weight"`
	Healthy     bool              `json:"healthy"`
	Metadata    map[string]string `json:"metadata"`
}

func NewServiceDiscovery(serverAddr, namespace string) (*ServiceDiscovery, error) {
	// 创建Nacos服务器配置
	sc := []constant.ServerConfig{
		*constant.NewServerConfig(serverAddr, 8848),
	}

	// 创建Nacos客户端配置
	cc := *constant.NewClientConfig(
		constant.WithUsername("nacos"),
		constant.WithPassword("nacos"),
		constant.WithNamespaceId(namespace),
		constant.WithTimeoutMs(5000),
		constant.WithNotLoadCacheAtStart(true),
		constant.WithLogDir("/tmp/nacos/log"),
		constant.WithCacheDir("/tmp/nacos/cache"),
		constant.WithLogLevel("info"),
	)

	// 创建服务发现客户端
	namingClient, err := clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &cc,
			ServerConfigs: sc,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create nacos client: %v", err)
	}

	sd := &ServiceDiscovery{
		client:        namingClient,
		services:      make(map[string][]ServiceInstance),
		stopChan:      make(chan struct{}),
		connectChan:   make(chan struct{}, 1),
		subscriptions: make(map[string]struct{}),
	}

	// 启动连接监控
	go sd.monitorConnection()

	return sd, nil
}

// monitorConnection 监控连接状态
func (sd *ServiceDiscovery) monitorConnection() {
	// 首次连接检查
	if sd.checkConnection() {
		sd.connected = true
		sd.connectChan <- struct{}{}
		log.Println("Nacos client connected initially")
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sd.stopChan:
			return
		case <-ticker.C:
			connected := sd.checkConnection()
			if connected && !sd.connected {
				sd.connected = true
				select {
				case sd.connectChan <- struct{}{}:
					log.Println("Nacos client connected")
				default:
				}
			} else if !connected && sd.connected {
				sd.connected = false
				log.Println("Nacos client disconnected")
			}
		}
	}
}

// checkConnection 检查连接状态 - 修正版本
func (sd *ServiceDiscovery) checkConnection() bool {
	// 方法1: 尝试获取一个不存在的服务，检查错误类型
	_, err := sd.client.GetService(vo.GetServiceParam{
		ServiceName: "__nacos_connection_check__",
		GroupName:   "DEFAULT_GROUP",
	})

	// 如果没有错误或者错误不是连接相关的，认为连接正常
	if err == nil {
		return true
	}

	// 检查错误信息，如果是连接相关的错误，返回false
	errMsg := strings.ToLower(err.Error())
	if strings.Contains(errMsg, "not connected") ||
		strings.Contains(errMsg, "connection") ||
		strings.Contains(errMsg, "client is starting") {
		return false
	}

	// 其他错误（如服务不存在）认为连接正常
	return true
}

// WaitForConnection 等待连接建立
func (sd *ServiceDiscovery) WaitForConnection(timeout time.Duration) error {
	if sd.connected {
		return nil
	}

	// 设置轮询间隔
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	timeoutChan := time.After(timeout)

	for {
		select {
		case <-timeoutChan:
			return fmt.Errorf("wait for nacos connection timeout after %v", timeout)
		case <-ticker.C:
			if sd.checkConnection() {
				sd.connected = true
				return nil
			}
		}
	}
}

// SubscribeService 订阅服务变化
func (sd *ServiceDiscovery) SubscribeService(serviceName string) error {
	// 等待客户端连接成功
	if err := sd.WaitForConnection(15 * time.Second); err != nil {
		return fmt.Errorf("nacos client not connected: %v", err)
	}

	callback := func(services []model.Instance, err error) {
		if err != nil {
			log.Printf("Service discovery callback error for %s: %v", serviceName, err)
			return
		}

		sd.updateServiceInstances(serviceName, services)
	}

	err := sd.client.Subscribe(&vo.SubscribeParam{
		ServiceName:       serviceName,
		Clusters:          []string{"DEFAULT"},
		GroupName:         "DEFAULT_GROUP",
		SubscribeCallback: callback,
	})

	if err != nil {
		return fmt.Errorf("failed to subscribe service %s: %v", serviceName, err)
	}

	sd.mu.Lock()
	sd.subscriptions[serviceName] = struct{}{}
	sd.mu.Unlock()

	// 初始获取服务实例
	go sd.refreshService(serviceName)

	log.Printf("Successfully subscribed to service: %s", serviceName)
	return nil
}

// updateServiceInstances 更新服务实例列表
func (sd *ServiceDiscovery) updateServiceInstances(serviceName string, instances []model.Instance) {
	healthyInstances := make([]ServiceInstance, 0, len(instances))
	for _, instance := range instances {
		if instance.Healthy {
			healthyInstances = append(healthyInstances, ServiceInstance{
				InstanceId:  instance.InstanceId,
				ServiceName: instance.ServiceName,
				IP:          instance.Ip,
				Port:        instance.Port,
				Weight:      instance.Weight,
				Healthy:     instance.Healthy,
				Metadata:    instance.Metadata,
			})
		}
	}

	sd.mu.Lock()
	defer sd.mu.Unlock()

	oldCount := len(sd.services[serviceName])
	sd.services[serviceName] = healthyInstances

	if oldCount != len(healthyInstances) {
		log.Printf("Service %s instances changed: %d -> %d healthy instances",
			serviceName, oldCount, len(healthyInstances))
	}
}

// refreshService 刷新服务实例
func (sd *ServiceDiscovery) refreshService(serviceName string) {
	// 重试机制
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		instances, err := sd.client.SelectInstances(vo.SelectInstancesParam{
			ServiceName: serviceName,
			Clusters:    []string{"DEFAULT"},
			GroupName:   "DEFAULT_GROUP",
			HealthyOnly: true,
		})
		if err != nil {
			log.Printf("Failed to refresh service %s (attempt %d/%d): %v", serviceName, i+1, maxRetries, err)
			if i < maxRetries-1 {
				time.Sleep(time.Duration(i+1) * time.Second)
				continue
			}
			return
		}

		sd.updateServiceInstances(serviceName, instances)
		log.Printf("Service %s refreshed, healthy instances: %d", serviceName, len(instances))
		break
	}
}

// UnsubscribeService 取消订阅服务
func (sd *ServiceDiscovery) UnsubscribeService(serviceName string) error {
	err := sd.client.Unsubscribe(&vo.SubscribeParam{
		ServiceName: serviceName,
		Clusters:    []string{"DEFAULT"},
		GroupName:   "DEFAULT_GROUP",
	})

	if err != nil {
		return fmt.Errorf("failed to unsubscribe service %s: %v", serviceName, err)
	}

	sd.mu.Lock()
	defer sd.mu.Unlock()

	delete(sd.services, serviceName)
	delete(sd.subscriptions, serviceName)

	log.Printf("Successfully unsubscribed from service: %s", serviceName)
	return nil
}

// GetServiceInstance 获取服务实例（负载均衡）
func (sd *ServiceDiscovery) GetServiceInstance(serviceName string) (*ServiceInstance, error) {
	sd.mu.RLock()
	instances, exists := sd.services[serviceName]
	sd.mu.RUnlock()

	if !exists || len(instances) == 0 {
		// 如果没有缓存，尝试直接查询
		return sd.getServiceInstanceDirectly(serviceName)
	}

	// 简单的随机负载均衡
	rand.New(rand.NewSource(time.Now().UnixNano()))
	instance := instances[rand.Intn(len(instances))]

	return &instance, nil
}

// getServiceInstanceDirectly 直接查询服务实例
func (sd *ServiceDiscovery) getServiceInstanceDirectly(serviceName string) (*ServiceInstance, error) {
	instances, err := sd.client.SelectInstances(vo.SelectInstancesParam{
		ServiceName: serviceName,
		Clusters:    []string{"DEFAULT"},
		GroupName:   "DEFAULT_GROUP",
		HealthyOnly: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get instances for service %s: %v", serviceName, err)
	}

	if len(instances) == 0 {
		return nil, fmt.Errorf("no healthy instances found for service: %s", serviceName)
	}

	// 随机选择一个实例
	rand.New(rand.NewSource(time.Now().UnixNano()))
	instance := instances[rand.Intn(len(instances))]

	return &ServiceInstance{
		InstanceId:  instance.InstanceId,
		ServiceName: instance.ServiceName,
		IP:          instance.Ip,
		Port:        instance.Port,
		Weight:      instance.Weight,
		Healthy:     instance.Healthy,
		Metadata:    instance.Metadata,
	}, nil
}

// GetServiceURL 获取服务URL
func (sd *ServiceDiscovery) GetServiceURL(serviceName string) (string, error) {
	instance, err := sd.GetServiceInstance(serviceName)
	if err != nil {
		return "", err
	}

	// 检查是否使用HTTPS
	scheme := "http"
	if instance.Metadata["https"] == "true" {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s:%d", scheme, instance.IP, instance.Port), nil
}

// GetAllInstances 获取所有服务实例
func (sd *ServiceDiscovery) GetAllInstances(serviceName string) ([]ServiceInstance, error) {
	sd.mu.RLock()
	instances, exists := sd.services[serviceName]
	sd.mu.RUnlock()

	if exists {
		// 返回副本，避免外部修改
		result := make([]ServiceInstance, len(instances))
		copy(result, instances)
		return result, nil
	}

	// 如果没有缓存，直接查询
	return sd.getAllInstancesDirectly(serviceName)
}

// getAllInstancesDirectly 直接获取所有实例
func (sd *ServiceDiscovery) getAllInstancesDirectly(serviceName string) ([]ServiceInstance, error) {
	instances, err := sd.client.SelectInstances(vo.SelectInstancesParam{
		ServiceName: serviceName,
		Clusters:    []string{"DEFAULT"},
		GroupName:   "DEFAULT_GROUP",
		HealthyOnly: true,
	})
	if err != nil {
		return nil, err
	}

	result := make([]ServiceInstance, 0, len(instances))
	for _, instance := range instances {
		result = append(result, ServiceInstance{
			InstanceId:  instance.InstanceId,
			ServiceName: instance.ServiceName,
			IP:          instance.Ip,
			Port:        instance.Port,
			Weight:      instance.Weight,
			Healthy:     instance.Healthy,
			Metadata:    instance.Metadata,
		})
	}

	return result, nil
}

// IsServiceAvailable 检查服务是否可用
func (sd *ServiceDiscovery) IsServiceAvailable(serviceName string) bool {
	sd.mu.RLock()
	instances, exists := sd.services[serviceName]
	sd.mu.RUnlock()

	return exists && len(instances) > 0
}

// GetAvailableServices 获取所有已订阅的服务名称
func (sd *ServiceDiscovery) GetAvailableServices() []string {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	services := make([]string, 0, len(sd.services))
	for serviceName := range sd.services {
		services = append(services, serviceName)
	}
	return services
}

// DeregisterInstance 注销实例
func (sd *ServiceDiscovery) DeregisterInstance(serviceName, ip string, port uint64) error {
	_, err := sd.client.DeregisterInstance(vo.DeregisterInstanceParam{
		ServiceName: serviceName,
		Ip:          ip,
		Port:        port,
		Cluster:     "DEFAULT",
		GroupName:   "DEFAULT_GROUP",
		Ephemeral:   true,
	})
	return err
}

// RegisterInstance 注册实例
func (sd *ServiceDiscovery) RegisterInstance(serviceName, ip string, port uint64, metadata map[string]string) error {
	_, err := sd.client.RegisterInstance(vo.RegisterInstanceParam{
		ServiceName: serviceName,
		Ip:          ip,
		Port:        port,
		Weight:      1.0,
		Enable:      true,
		Healthy:     true,
		Metadata:    metadata,
		ClusterName: "DEFAULT",
		GroupName:   "DEFAULT_GROUP",
		Ephemeral:   true,
	})
	return err
}

// Close 关闭服务发现
func (sd *ServiceDiscovery) Close() {
	close(sd.stopChan)

	// 取消所有订阅
	sd.mu.RLock()
	subscriptions := make([]string, 0, len(sd.subscriptions))
	for serviceName := range sd.subscriptions {
		subscriptions = append(subscriptions, serviceName)
	}
	sd.mu.RUnlock()

	for _, serviceName := range subscriptions {
		if err := sd.UnsubscribeService(serviceName); err != nil {
			log.Printf("Failed to unsubscribe service %s: %v", serviceName, err)
		}
	}

	log.Println("Service discovery closed")
}
