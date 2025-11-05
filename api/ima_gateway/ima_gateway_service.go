package ima_gateway

import (
	"bdsw-im-ws/api/common"
	"context"
)

// ================================
// IMA 网关服务接口
// ================================

type IMAGatewayService interface {
	// 单用户推送
	PushToUser(ctx context.Context, req *PushRequest) (*common.BaseResponse, error)
	// 批量推送
	PushToUsers(ctx context.Context, req *BatchPushRequest) (*common.BaseResponse, error)
	// 广播
	Broadcast(ctx context.Context, req *BroadcastMessage) (*common.BaseResponse, error)
	// 踢用户下线
	KickUser(ctx context.Context, req *KickUserRequest) (*common.BaseResponse, error)
	// 获取在线用户列表
	GetOnlineUsers(ctx context.Context, req *GetOnlineUsersRequest) (*GetOnlineUsersResponse, error)
	// 检查用户是否在线
	CheckUserOnline(ctx context.Context, req *CheckUserOnlineRequest) (*CheckUserOnlineResponse, error)
}
