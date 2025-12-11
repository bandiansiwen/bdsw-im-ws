package service

import (
	"context"
	"fmt"
	"log"
	"slices"
	"sync"
	"time"

	"github.com/bdsw/bdsw-im-ws/internal/config"
	"github.com/bdsw/bdsw-im-ws/internal/dubbo/consumer"
	"github.com/bdsw/bdsw-im-ws/proto/common"
	dubboservice "github.com/bdsw/bdsw-im-ws/proto/service"

	"github.com/bdsw/bdsw-im-ws/internal/redis"
	"github.com/bdsw/bdsw-im-ws/pkg/utils"

	"github.com/gorilla/websocket"
)

type GatewayService struct {
	instanceID  string
	redisClient *redis.Client
	dubboClient *consumer.IMAServiceConsumer

	// 连接管理
	userConnections *UserConnectionManager
	websocketServer *WebSocketServer

	// 服务状态
	started bool
	mu      sync.RWMutex

	// 关闭通道
	shutdownChan chan struct{}
}

func NewGatewayService(redisClient *redis.Client) *GatewayService {
	instanceID := utils.GenerateInstanceID()

	return &GatewayService{
		instanceID:      instanceID,
		redisClient:     redisClient,
		userConnections: NewUserConnectionManager(),
		websocketServer: NewWebSocketServer(&WebSocketConfig{
			Port:            config.GetConfig().WebSocket.Server.Port,
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}),
		shutdownChan: make(chan struct{}),
	}
}

func (s *GatewayService) SetDubboClient(client *consumer.IMAServiceConsumer) {
	s.dubboClient = client
}

func (s *GatewayService) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return fmt.Errorf("service already started")
	}

	// 启动 WebSocket 服务器
	go s.websocketServer.Start(s.handleWebSocketConnection)

	// 启动统计报告
	go s.startStatsReporter()

	s.started = true
	log.Printf("IMA Gateway started with instance ID: %s", s.instanceID)
	return nil
}

func (s *GatewayService) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return nil
	}

	log.Println("Shutting down IMA Gateway...")

	// 关闭信号通道
	close(s.shutdownChan)

	// 2. 关闭所有 WebSocket 连接
	s.userConnections.CloseAll()

	// 3. 清理 Redis 注册信息
	s.cleanupRedisRegistrations()

	s.started = false
	log.Println("IMA Gateway shutdown completed")
	return nil
}

// 确保不复制 websocket.Conn
func (s *GatewayService) handleWebSocketConnection(conn *websocket.Conn, userID string, token string, deviceID string) {
	log.Printf("WebSocket connection attempt for user: %s, device: %s", userID, deviceID)

	// TODO: 1. 调用逻辑网关进行 Token 验证, 检查是否多端登录

	// 2. 注册用户连接 - 传递指针
	s.userConnections.Register(userID, deviceID, conn)

	// 3. 注册到 Redis
	s.registerUserConnection(userID, deviceID)

	// 4. 通知业务服务用户上线
	s.notifyUserStatusChange(userID, deviceID, "online")

	log.Printf("User %s (device: %s) connected successfully", userID, deviceID)

	defer func() {
		// 连接关闭时的清理
		s.userConnections.Unregister(userID, deviceID)
		s.unregisterUserConnection(userID, deviceID)

		// 通知业务服务用户下线
		s.notifyUserStatusChange(userID, deviceID, "offline")

		conn.Close()
		log.Printf("WebSocket connection closed for user: %s, device: %s", userID, deviceID)
	}()

	// 消息处理循环
	for {
		select {
		case <-s.shutdownChan:
			return
		default:
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket read error for user %s: %v", userID, err)
				}
				return
			}

			if messageType == websocket.TextMessage {
				s.handleClientMessage(userID, deviceID, message)
			}
		}
	}
}

func (s *GatewayService) kickExistingConnection(userID, deviceID, reason string) {
	if conn := s.userConnections.Get(userID, deviceID); conn != nil {
		// 发送被踢下线的消息
		kickMsg := &common.Message{
			MsgId: utils.GenerateMessageID(),
			Type:  common.MessageType_TEXT,
			Content: &common.MessageContent{
				Content: &common.MessageContent_Text{
					Text: &common.TextContent{
						Text: reason,
					},
				},
			},
		}

		conn.WriteJSON(kickMsg)
		conn.Close()

		s.userConnections.Unregister(userID, deviceID)
		s.unregisterUserConnection(userID, deviceID)

		log.Printf("Kicked user %s (device: %s) for reason: %s", userID, deviceID, reason)
	}
}

func (s *GatewayService) handleClientMessage(userID, deviceID string, rawMessage []byte) {
	startTime := time.Now()

	// 直接转发到业务服务
	serverMsg, err := s.dubboClient.GatewayService.HandleClientMessage(
		context.Background(),
		&dubboservice.ClientMessageRequest{
			Message: rawMessage,
		},
	)
	if err != nil {
		log.Printf("Failed to process message for user %s: %v", userID, err)
		s.sendErrorToUser(userID, deviceID, "Service temporarily unavailable")
		return
	}

	// 如果有立即响应，直接返回给用户
	if serverMsg != nil {
		msg := &common.Message{
			MsgId:      serverMsg.MsgId,
			Status:     serverMsg.Status,
			ServerTime: serverMsg.ServerTime,
			Content: &common.MessageContent{
				Content: &common.MessageContent_Text{
					Text: &common.TextContent{
						Text: serverMsg.ErrorMsg,
					},
				},
			},
		}
		if err := s.sendServerMessageToUser(userID, deviceID, msg); err != nil {
			log.Printf("Failed to send response to user %s: %v", userID, err)
		}
	}

	// 记录处理时间
	duration := time.Since(startTime)
	if duration > 100*time.Millisecond {
		log.Printf("Slow message processing for user %s: %v", userID, duration)
	}
}

// 确保所有 WebSocket 操作都使用指针
func (s *GatewayService) sendServerMessageToUser(userID, deviceID string, serverMsg *common.Message) error {
	conn := s.userConnections.Get(userID, deviceID)
	if conn == nil {
		return fmt.Errorf("user %s (device: %s) not connected", userID, deviceID)
	}

	// 使用指针操作，不复制连接
	return conn.WriteJSON(serverMsg)
}

func (s *GatewayService) sendErrorToUser(userID, deviceID, errorMsg string) {
	conn := s.userConnections.Get(userID, deviceID)
	if conn != nil {
		errorResponse := &common.Message{
			MsgId:            utils.GenerateMessageID(),
			Type:             common.MessageType_TEXT,
			ConversationType: common.ConversationType_SYSTEM_CHAT,
			ServerTime:       time.Now().Unix(),
			Content: &common.MessageContent{
				Content: &common.MessageContent_Text{
					Text: &common.TextContent{
						Text: errorMsg,
					},
				},
			},
		}

		conn.WriteJSON(errorResponse)
	}
}

func (s *GatewayService) notifyUserStatusChange(userID, deviceID, status string) {
	// 这里可以扩展为向业务服务发送状态变更通知
	log.Printf("User %s (device: %s) status changed to: %s", userID, deviceID, status)
}

func (s *GatewayService) registerUserConnection(userID, deviceID string) {
	connInfo := map[string]interface{}{
		"user_id":      userID,
		"device_id":    deviceID,
		"ima_instance": s.instanceID,
		"login_time":   time.Now().Unix(),
		"last_active":  time.Now().Unix(),
	}

	if err := s.redisClient.SetUserConnection(userID, deviceID, connInfo); err != nil {
		log.Printf("Failed to register user %s in Redis: %v", userID, err)
	}
}

func (s *GatewayService) unregisterUserConnection(userID, deviceID string) {
	if err := s.redisClient.RemoveUserConnection(userID, deviceID); err != nil {
		log.Printf("Failed to unregister user %s from Redis: %v", userID, err)
	}
}

func (s *GatewayService) cleanupRedisRegistrations() {
	// 清理所有本实例的用户连接
	userDevices := s.userConnections.GetAllUserDevices()
	for userID, devices := range userDevices {
		for _, deviceID := range devices {
			s.unregisterUserConnection(userID, deviceID)
		}
	}

	// 清理实例状态
	if err := s.redisClient.RemoveInstanceStatus(s.instanceID); err != nil {
		log.Printf("Failed to cleanup instance status: %v", err)
	}
}

func (s *GatewayService) startStatsReporter() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.shutdownChan:
			return
		case <-ticker.C:
			onlineCount := s.userConnections.GetOnlineCount()
			log.Printf("Gateway stats - Online users: %d, Instance: %s", onlineCount, s.instanceID)
		}
	}
}

// PushToTarget 实现单用户推送
func (s *GatewayService) PushToTarget(ctx context.Context, req *common.PushRequest) (*common.PushResponse, error) {

	targets := req.GetTargetDevices().GetDevices()

	if targets == nil || len(targets) == 0 {
		log.Printf("Gateway PushToUser - no device ids specified")
		return &common.PushResponse{
			Success: false,
		}, nil
	}

	totalCount := len(targets)

	successCount := 0
	var res []*common.PushResult
	for i := 0; i < totalCount; i++ {
		target := targets[i]
		if err := s.sendServerMessageToUser(target.GetUserId(), target.GetDeviceId(), req.Message); err != nil {
			item := &common.PushResult{
				Status:   3,
				PushTime: time.Now().Unix(),
			}
			res[i] = item
		} else {
			successCount++
		}
	}

	return &common.PushResponse{
		Success:      true,
		TotalTargets: int32(totalCount),
		SuccessCount: int32(successCount),
		FailedCount:  int32(totalCount - successCount),
		Results:      res,
	}, nil
}

// Broadcast 实现广播消息
func (s *GatewayService) Broadcast(ctx context.Context, req *common.PushBroadcastRequest) (*dubboservice.WsBroadcastResponse, error) {

	serverMsg := req.Message
	totalCount := s.userConnections.GetOnlineCount()
	errorCount := 0
	s.userConnections.Broadcast(func(userID, deviceID string, conn *websocket.Conn) {
		if err := conn.WriteJSON(serverMsg); err != nil {
			errorCount++
			log.Printf("Failed to broadcast to user %s (device: %s): %v", userID, deviceID, err)
		}
	})

	return &dubboservice.WsBroadcastResponse{
		TotalUsers:  int64(totalCount),
		PushedUsers: int64(totalCount - errorCount),
		FailedUsers: int64(errorCount),
		TaskId:      req.RequestId,
	}, nil
}

// KickUser 实现踢用户下线
func (s *GatewayService) KickUser(ctx context.Context, req *dubboservice.WsKickUserRequest) (*common.Response, error) {

	targets := req.GetTargetDevices().GetDevices()
	if targets == nil || len(targets) == 0 {
		log.Printf("Gateway PushToUser - no device ids specified")
		return &common.Response{
			Code: 500,
		}, nil
	}

	excludeDeviceIds := req.GetExcludeDeviceIds()

	totalCount := len(targets)
	for i := 0; i < totalCount; i++ {
		target := targets[i]
		// 排除
		if excludeDeviceIds != nil && slices.Contains(excludeDeviceIds, target.DeviceId) {
			continue
		}
		s.kickExistingConnection(target.UserId, target.DeviceId, req.ReasonMessage)
	}

	return &common.Response{
		Code:      200,
		Timestamp: time.Now().Unix(),
	}, nil
}
