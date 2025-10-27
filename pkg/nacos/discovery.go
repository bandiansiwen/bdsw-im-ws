package nacos

import (
	_ "context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

type ServiceDiscovery struct {
	client   naming_client.INamingClient
	services map[string][]ServiceInstance
	mu       sync.RWMutex
	stopChan chan struct{}
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
		client:   namingClient,
		services: make(map[string][]ServiceInstance),
		stopChan: make(chan struct{}),
	}

	return sd, nil
}

// SubscribeService 订阅服务变化
func (sd *ServiceDiscovery) SubscribeService(serviceName string) error {
	err := sd.client.Subscribe(&vo.SubscribeParam{
		ServiceName: serviceName,
		Clusters:    []string{"DEFAULT"},
		GroupName:   "DEFAULT_GROUP",
		SubscribeCallback: func(services []model.Instance, err error) {
			if err != nil {
				log.Printf("Service discovery callback error for %s: %v", serviceName, err)
				return
			}

			sd.mu.Lock()
			defer sd.mu.Unlock()

			instances := make([]ServiceInstance, 0, len(services))
			for _, service := range services {
				if service.Healthy {
					instances = append(instances, ServiceInstance{
						InstanceId:  service.InstanceId,
						ServiceName: service.ServiceName,
						IP:          service.Ip,
						Port:        service.Port,
						Weight:      service.Weight,
						Healthy:     service.Healthy,
						Metadata:    service.Metadata,
					})
				}
			}

			sd.services[serviceName] = instances
			log.Printf("Service %s updated, healthy instances: %d", serviceName, len(instances))
		},
	})

	if err != nil {
		return fmt.Errorf("failed to subscribe service %s: %v", serviceName, err)
	}

	// 初始获取服务实例
	go sd.refreshService(serviceName)

	return nil
}

// refreshService 刷新服务实例
func (sd *ServiceDiscovery) refreshService(serviceName string) {
	instances, err := sd.client.SelectInstances(vo.SelectInstancesParam{
		ServiceName: serviceName,
		Clusters:    []string{"DEFAULT"},
		GroupName:   "DEFAULT_GROUP",
		HealthyOnly: true,
	})
	if err != nil {
		log.Printf("Failed to refresh service %s: %v", serviceName, err)
		return
	}

	sd.mu.Lock()
	defer sd.mu.Unlock()

	serviceInstances := make([]ServiceInstance, 0, len(instances))
	for _, instance := range instances {
		serviceInstances = append(serviceInstances, ServiceInstance{
			InstanceId:  instance.InstanceId,
			ServiceName: instance.ServiceName,
			IP:          instance.Ip,
			Port:        instance.Port,
			Weight:      instance.Weight,
			Healthy:     instance.Healthy,
			Metadata:    instance.Metadata,
		})
	}

	sd.services[serviceName] = serviceInstances
	log.Printf("Service %s refreshed, healthy instances: %d", serviceName, len(serviceInstances))
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
		return instances, nil
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

// DeregisterInstance 注销实例
func (sd *ServiceDiscovery) DeregisterInstance(serviceName, ip string, port uint64) error {
	_, err := sd.client.DeregisterInstance(vo.DeregisterInstanceParam{
		ServiceName: serviceName,
		Ip:          ip,
		Port:        port,
		Cluster:     "DEFAULT",
		GroupName:   "DEFAULT_GROUP",
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
}
