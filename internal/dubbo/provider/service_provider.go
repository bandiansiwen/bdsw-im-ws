package provider

import (
	"context"
	"log"

	"dubbo.apache.org/dubbo-go/v3/config"
	"github.com/bdsw/bdsw-im-ws/api/common"

	"github.com/bdsw/bdsw-im-ws/api/ima"
	"github.com/bdsw/bdsw-im-ws/internal/service"
)

// IMAServiceProvider 实现 Dubbo 服务，供业务服务调用
type IMAServiceProvider struct {
	ima.UnimplementedImaServiceServer
	gatewayService *service.GatewayService
}

func NewIMAServiceProvider(gatewayService *service.GatewayService) *IMAServiceProvider {
	server := &IMAServiceProvider{
		gatewayService: gatewayService,
	}
	config.SetProviderService(server)
	return server
}

// PushToUser 单用户推送
func (p *IMAServiceProvider) PushToUser(ctx context.Context, req *ima.PushRequest) (*common.BaseResponse, error) {
	log.Printf("Received push request for user: %s, device: %s", req.UserId, req.DeviceId)
	return p.gatewayService.PushToUser(ctx, req)
}

// PushToUsers 批量推送
func (p *IMAServiceProvider) PushToUsers(ctx context.Context, req *ima.BatchPushRequest) (*common.BaseResponse, error) {
	log.Printf("Received batch push request for %d users", len(req.Pushes))
	return p.gatewayService.PushToUsers(ctx, req)
}

// Broadcast 广播消息
func (p *IMAServiceProvider) Broadcast(ctx context.Context, req *ima.BroadcastMessage) (*common.BaseResponse, error) {
	log.Printf("Received broadcast request")
	return p.gatewayService.Broadcast(ctx, req)
}

// KickUser 踢用户下线
func (p *IMAServiceProvider) KickUser(ctx context.Context, req *ima.KickUserRequest) (*common.BaseResponse, error) {
	log.Printf("Received kick request for user: %s, device: %s, reason: %s", req.UserId, req.DeviceId, req.Reason)
	return p.gatewayService.KickUser(ctx, req)
}
