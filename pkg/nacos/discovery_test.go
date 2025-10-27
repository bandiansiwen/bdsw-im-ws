package nacos

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServiceInstance(t *testing.T) {
	instance := ServiceInstance{
		InstanceId:  "test-instance",
		ServiceName: "test-service",
		IP:          "127.0.0.1",
		Port:        8080,
		Weight:      1.0,
		Healthy:     true,
		Metadata:    map[string]string{"key": "value"},
	}

	assert.Equal(t, "test-instance", instance.InstanceId)
	assert.Equal(t, "test-service", instance.ServiceName)
	assert.Equal(t, "127.0.0.1", instance.IP)
	assert.Equal(t, uint64(8080), instance.Port)
	assert.Equal(t, 1.0, instance.Weight)
	assert.True(t, instance.Healthy)
	assert.Equal(t, "value", instance.Metadata["key"])
}

func TestNewServiceDiscovery(t *testing.T) {
	// 测试创建服务发现客户端（会失败，因为没有真实的Nacos服务器）
	_, err := NewServiceDiscovery("invalid-host:8848", "test")

	// 在测试环境中，我们期望创建失败
	assert.Error(t, err)

	// 测试错误消息包含相关信息
	assert.Contains(t, err.Error(), "failed to create nacos client")
}

func TestServiceDiscovery_Mock(t *testing.T) {
	// 创建一个模拟的ServiceDiscovery用于测试方法
	sd := &ServiceDiscovery{
		services: make(map[string][]ServiceInstance),
		stopChan: make(chan struct{}),
	}

	// 测试空服务的情况
	instance, err := sd.GetServiceInstance("nonexistent-service")
	assert.Error(t, err)
	assert.Nil(t, instance)

	// 测试获取服务URL
	url, err := sd.GetServiceURL("nonexistent-service")
	assert.Error(t, err)
	assert.Equal(t, "", url)

	// 测试获取所有实例
	instances, err := sd.GetAllInstances("nonexistent-service")
	assert.Error(t, err)
	assert.Nil(t, instances)

	// 测试关闭
	sd.Close()
	// 验证stopChan已关闭
	select {
	case <-sd.stopChan:
		// 正常关闭
	default:
		t.Error("stopChan should be closed")
	}
}
