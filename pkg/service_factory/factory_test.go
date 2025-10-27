package service_factory

import (
	"testing"

	"bdsw-im-ws/internal/config"
	"bdsw-im-ws/pkg/nacos"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockDiscovery 模拟服务发现
type MockDiscovery struct {
	services map[string][]nacos.ServiceInstance
}

func NewMockDiscovery() *MockDiscovery {
	return &MockDiscovery{
		services: make(map[string][]nacos.ServiceInstance),
	}
}

func (m *MockDiscovery) SubscribeService(serviceName string) error {
	// 模拟订阅服务
	m.services[serviceName] = []nacos.ServiceInstance{
		{
			InstanceId:  "mock-instance",
			ServiceName: serviceName,
			IP:          "127.0.0.1",
			Port:        8080,
			Weight:      1.0,
			Healthy:     true,
			Metadata:    map[string]string{},
		},
	}
	return nil
}

func (m *MockDiscovery) GetServiceInstance(serviceName string) (*nacos.ServiceInstance, error) {
	instances, exists := m.services[serviceName]
	if !exists || len(instances) == 0 {
		return nil, assert.AnError
	}
	return &instances[0], nil
}

func (m *MockDiscovery) GetServiceURL(serviceName string) (string, error) {
	instance, err := m.GetServiceInstance(serviceName)
	if err != nil {
		return "", err
	}
	return "http://" + instance.IP + ":8080", nil
}

func (m *MockDiscovery) GetAllInstances(serviceName string) ([]nacos.ServiceInstance, error) {
	instances, exists := m.services[serviceName]
	if !exists {
		return nil, assert.AnError
	}
	return instances, nil
}

func (m *MockDiscovery) DeregisterInstance(serviceName, ip string, port uint64) error {
	return nil
}

func (m *MockDiscovery) RegisterInstance(serviceName, ip string, port uint64, metadata map[string]string) error {
	return nil
}

func (m *MockDiscovery) Close() {}

func TestServiceFactory_WithMock(t *testing.T) {
	cfg := &config.Config{
		Env: "test",
		Nacos: struct {
			ServerAddr string `mapstructure:"NACOS_SERVER_ADDR"`
			Namespace  string `mapstructure:"NACOS_NAMESPACE"`
			Group      string `mapstructure:"NACOS_GROUP"`
			DataID     string `mapstructure:"NACOS_DATA_ID"`
		}{
			ServerAddr: "localhost:8848",
			Namespace:  "test",
		},
		ServiceNames: struct {
			UserService     string `mapstructure:"USER_SERVICE_NAME"`
			BusinessService string `mapstructure:"BUSINESS_SERVICE_NAME"`
		}{
			UserService:     "user-service-test",
			BusinessService: "business-service-test",
		},
	}

	// 创建模拟服务工厂
	factory := &ServiceFactory{
		discovery:   NewMockDiscovery(),
		cfg:         cfg,
		initialized: false,
	}

	// 测试初始化
	err := factory.Init()
	require.NoError(t, err)
	assert.True(t, factory.initialized)

	// 测试服务可用性检查
	assert.True(t, factory.HasAuthService())
	assert.True(t, factory.HasBusinessService())

	// 测试获取客户端
	assert.NotNil(t, factory.GetAuthClient())
	assert.NotNil(t, factory.GetBusinessClient())

	// 测试关闭
	factory.Close()
}

func TestServiceFactory_NoServices(t *testing.T) {
	cfg := &config.Config{
		Env: "test",
		ServiceNames: struct {
			UserService     string `mapstructure:"USER_SERVICE_NAME"`
			BusinessService string `mapstructure:"BUSINESS_SERVICE_NAME"`
		}{
			UserService:     "", // 空的服务名
			BusinessService: "",
		},
	}

	factory := &ServiceFactory{
		discovery:   NewMockDiscovery(),
		cfg:         cfg,
		initialized: false,
	}

	err := factory.Init()
	require.NoError(t, err)

	// 应该没有可用的服务
	assert.False(t, factory.HasAuthService())
	assert.False(t, factory.HasBusinessService())
	assert.Nil(t, factory.GetAuthClient())
	assert.Nil(t, factory.GetBusinessClient())
}
