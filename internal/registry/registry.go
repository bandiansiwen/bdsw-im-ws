package registry

import (
	"bdsw-im-ws/internal/dubbo/client"
	"bdsw-im-ws/internal/dubbo/provider"
	"bdsw-im-ws/internal/redis"
	"bdsw-im-ws/internal/service"
	"context"

	"dubbo.apache.org/dubbo-go/v3/config"
)

// ServiceRegistry 服务注册器
type ServiceRegistry struct {
	GatewayService  *service.GatewayService
	ServiceProvider *provider.IMAGatewayServiceProvider
}

// NewServiceRegistry 创建服务注册器
func NewServiceRegistry(redisClient *redis.Client) (*ServiceRegistry, error) {
	// 创建网关服务
	gatewayService := service.NewGatewayService(redisClient)

	// 创建 Dubbo 客户端
	dubboClient, err := client.NewClient()
	if err != nil {
		return nil, err
	}
	gatewayService.SetDubboClient(dubboClient)

	// 创建服务提供者
	serviceProvider := provider.NewIMAGatewayServiceProvider(gatewayService)

	return &ServiceRegistry{
		GatewayService:  gatewayService,
		ServiceProvider: serviceProvider,
	}, nil
}

// RegisterDubboServices 注册 Dubbo 服务
func (r *ServiceRegistry) RegisterDubboServices() {
	config.SetProviderService(r.ServiceProvider)
}

// StartAll 启动所有服务
func (r *ServiceRegistry) StartAll() error {
	return r.GatewayService.Start()
}

// ShutdownAll 关闭所有服务
func (r *ServiceRegistry) ShutdownAll(ctx context.Context) error {
	return r.GatewayService.Shutdown(ctx)
}
