package server_push

import (
	"bdsw-im-ws/api/common"
	"bdsw-im-ws/api/ima_gateway"
	"context"
)

// ================================
// 服务端推送服务接口
// ================================

type ServerPushService interface {
	// 注册推送流
	RegisterPushStream(ctx context.Context) (ServerPushStreamClient, error)
	// 批量推送消息
	BatchPushMessages(ctx context.Context, req *ima_gateway.BatchPushRequest) (*common.BaseResponse, error)
	// 广播系统消息
	BroadcastSystemMessage(ctx context.Context, req *BroadcastSystemMessageRequest) (*common.BaseResponse, error)
	// 群组推送
	PushToGroup(ctx context.Context, req *GroupPushRequest) (*common.BaseResponse, error)
	// 强制刷新
	ForceRefresh(ctx context.Context, req *ForceRefreshRequest) (*common.BaseResponse, error)
}

type ServerPushStreamClient interface {
	// Send 发送心跳
	Send(heartbeat *Heartbeat) error
	// Recv 接收推送请求
	Recv() (*ima_gateway.PushRequest, error)
	// CloseSend 关闭发送
	CloseSend() error
}
