package service_factory

import (
	"log"
	"sync"

	"bdsw-im-ws/internal/config"
	"bdsw-im-ws/internal/service"
	"bdsw-im-ws/pkg/nacos"
)

type ServiceFactory struct {
	discovery      *nacos.ServiceDiscovery
	cfg            *config.Config
	authClient     *service.AuthClient
	businessClient *service.BusinessServiceClient
	mu             sync.RWMutex
	initialized    bool
}

func NewServiceFactory(cfg *config.Config) (*ServiceFactory, error) {
	discovery, err := nacos.NewServiceDiscovery(cfg.Nacos.ServerAddr, cfg.Nacos.Namespace)
	if err != nil {
		return nil, err
	}

	factory := &ServiceFactory{
		discovery:   discovery,
		cfg:         cfg,
		initialized: false,
	}

	return factory, nil
}

// Init 初始化服务客户端
func (f *ServiceFactory) Init() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.initialized {
		return nil
	}

	// 订阅用户服务
	if f.cfg.ServiceNames.UserService != "" {
		if err := f.discovery.SubscribeService(f.cfg.ServiceNames.UserService); err != nil {
			log.Printf("Failed to subscribe user service: %v", err)
		} else {
			f.authClient = service.NewAuthClient(f.discovery, f.cfg.ServiceNames.UserService)
			log.Printf("Auth client initialized for service: %s", f.cfg.ServiceNames.UserService)
		}
	}

	// 订阅业务服务
	if f.cfg.ServiceNames.BusinessService != "" {
		if err := f.discovery.SubscribeService(f.cfg.ServiceNames.BusinessService); err != nil {
			log.Printf("Failed to subscribe business service: %v", err)
		} else {
			f.businessClient = service.NewBusinessServiceClient(f.discovery, f.cfg.ServiceNames.BusinessService)
			log.Printf("Business client initialized for service: %s", f.cfg.ServiceNames.BusinessService)
		}
	}

	f.initialized = true
	return nil
}

// GetAuthClient 获取认证客户端
func (f *ServiceFactory) GetAuthClient() *service.AuthClient {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.authClient
}

// GetBusinessClient 获取业务客户端
func (f *ServiceFactory) GetBusinessClient() *service.BusinessServiceClient {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.businessClient
}

// HasAuthService 检查是否有认证服务
func (f *ServiceFactory) HasAuthService() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.authClient != nil
}

// HasBusinessService 检查是否有业务服务
func (f *ServiceFactory) HasBusinessService() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.businessClient != nil
}

// Close 关闭服务工厂
func (f *ServiceFactory) Close() {
	if f.discovery != nil {
		f.discovery.Close()
	}
}
