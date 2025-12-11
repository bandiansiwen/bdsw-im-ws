package provider

import (
	"context"
	"log"

	"dubbo.apache.org/dubbo-go/v3/config"
	"github.com/bdsw/bdsw-im-ws/internal/service"
	"github.com/bdsw/bdsw-im-ws/proto/common"
	dubboservice "github.com/bdsw/bdsw-im-ws/proto/service"
)

// IMAServiceProvider 实现 Dubbo 服务，供业务服务调用
type IMAServiceProvider struct {
	dubboservice.UnimplementedWsServiceServer
	gatewayService *service.GatewayService
}

func NewIMAServiceProvider(gatewayService *service.GatewayService) *IMAServiceProvider {
	server := &IMAServiceProvider{
		gatewayService: gatewayService,
	}
	config.SetProviderService(server)
	return server
}

// PushToTarget 推送
func (p *IMAServiceProvider) PushToTarget(ctx context.Context, req *common.PushRequest) (*common.PushResponse, error) {
	log.Printf("Received push request for target: %s", req.GetTargetDevices().GetDevices())
	return p.gatewayService.PushToTarget(ctx, req)
}

// KickTarget 踢用户下线
func (p *IMAServiceProvider) KickTarget(ctx context.Context, req *dubboservice.WsKickUserRequest) (*common.Response, error) {
	log.Printf("Received kick request for target: %s, reason: %s", req.GetTargetDevices().GetDevices(), req.GetReason())
	return p.gatewayService.KickUser(ctx, req)
}

// Broadcast 广播消息
func (p *IMAServiceProvider) Broadcast(ctx context.Context, req *common.PushBroadcastRequest) (*dubboservice.WsBroadcastResponse, error) {
	log.Printf("Received broadcast request")
	return p.gatewayService.Broadcast(ctx, req)
}
