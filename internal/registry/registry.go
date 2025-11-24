package registry

import (
	"context"
	"fmt"

	"github.com/bdsw/bdsw-im-ws/internal/dubbo/consumer"
	"github.com/bdsw/bdsw-im-ws/internal/dubbo/provider"
	"github.com/bdsw/bdsw-im-ws/internal/redis"
	"github.com/bdsw/bdsw-im-ws/internal/service"

	"dubbo.apache.org/dubbo-go/v3/config"
	_ "dubbo.apache.org/dubbo-go/v3/imports"
)

// ServiceRegistry 服务注册器
type ServiceRegistry struct {
	GatewayService  *service.GatewayService
	ServiceProvider *provider.IMAServiceProvider
}

// NewServiceRegistry 创建服务注册器
func NewServiceRegistry(redisClient *redis.Client) (*ServiceRegistry, error) {
	// 创建网关服务
	gatewayService := service.NewGatewayService(redisClient)

	// 创建 Dubbo 客户端
	dubboClient, err := consumer.NewIMAServiceConsumer()
	if err != nil {
		return nil, err
	}
	gatewayService.SetDubboClient(dubboClient)

	// 创建服务提供者
	serviceProvider := provider.NewIMAServiceProvider(gatewayService)

	return &ServiceRegistry{
		GatewayService:  gatewayService,
		ServiceProvider: serviceProvider,
	}, nil
}

// RegisterDubboServices 注册 Dubbo 服务
func (r *ServiceRegistry) RegisterDubboServices() {
	err := config.Load(config.WithPath("config/dubbo/dubbo.yml"))
	if err != nil {
		_ = fmt.Errorf("配置加载失败: %v", err)
	}
}

// StartAll 启动所有服务
func (r *ServiceRegistry) StartAll() error {
	return r.GatewayService.Start()
}

// ShutdownAll 关闭所有服务
func (r *ServiceRegistry) ShutdownAll(ctx context.Context) error {
	return r.GatewayService.Shutdown(ctx)
}
