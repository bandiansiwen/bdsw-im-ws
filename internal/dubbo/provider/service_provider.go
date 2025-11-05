package provider

import (
	"bdsw-im-ws/api/common"
	"context"
	"log"

	"bdsw-im-ws/api/ima_gateway"
	"bdsw-im-ws/internal/service"
)

// IMAGatewayServiceProvider 实现 Dubbo 服务，供业务服务调用
type IMAGatewayServiceProvider struct {
	gatewayService *service.GatewayService
}

func NewIMAGatewayServiceProvider(gatewayService *service.GatewayService) *IMAGatewayServiceProvider {
	return &IMAGatewayServiceProvider{
		gatewayService: gatewayService,
	}
}

// PushToUser 单用户推送
func (p *IMAGatewayServiceProvider) PushToUser(ctx context.Context, req *ima_gateway.PushRequest) (*common.BaseResponse, error) {
	log.Printf("Received push request for user: %s, device: %s", req.UserId, req.DeviceId)
	return p.gatewayService.PushToUser(ctx, req)
}

// PushToUsers 批量推送
func (p *IMAGatewayServiceProvider) PushToUsers(ctx context.Context, req *ima_gateway.BatchPushRequest) (*common.BaseResponse, error) {
	log.Printf("Received batch push request for %d users", len(req.Pushes))
	return p.gatewayService.PushToUsers(ctx, req)
}

// Broadcast 广播消息
func (p *IMAGatewayServiceProvider) Broadcast(ctx context.Context, req *ima_gateway.BroadcastMessage) (*common.BaseResponse, error) {
	log.Printf("Received broadcast request")
	return p.gatewayService.Broadcast(ctx, req)
}

// KickUser 踢用户下线
func (p *IMAGatewayServiceProvider) KickUser(ctx context.Context, req *ima_gateway.KickUserRequest) (*common.BaseResponse, error) {
	log.Printf("Received kick request for user: %s, device: %s, reason: %s", req.UserId, req.DeviceId, req.Reason)
	return p.gatewayService.KickUser(ctx, req)
}
