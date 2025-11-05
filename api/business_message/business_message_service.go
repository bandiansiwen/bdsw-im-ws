package business_message

import (
	"bdsw-im-ws/api/common"
	"context"
)

// ================================
// 业务消息服务接口
// ================================

type BusinessMessageService interface {
	// 处理客户端消息
	HandleClientMessage(ctx context.Context, req *common.ClientMessage) (*common.ServerMessage, error)
	// 处理批量消息
	HandleBatchMessages(ctx context.Context, req *HandleBatchMessagesRequest) (*HandleBatchMessagesResponse, error)
	// 获取用户会话列表
	GetUserSessions(ctx context.Context, req *GetUserSessionsRequest) (*GetUserSessionsResponse, error)
	// 获取聊天历史
	GetChatHistory(ctx context.Context, req *GetChatHistoryRequest) (*GetChatHistoryResponse, error)
	// 消息已读确认
	MarkMessageRead(ctx context.Context, req *MessageReadRequest) (*common.BaseResponse, error)
	// 撤回消息
	RecallMessage(ctx context.Context, req *RecallMessageRequest) (*common.BaseResponse, error)
}
