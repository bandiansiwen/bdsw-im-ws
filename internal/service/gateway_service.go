package service

import (
	"bdsw-im-ws/api/common"
	"bdsw-im-ws/api/ima_gateway"
	"bdsw-im-ws/api/muc"
	"bdsw-im-ws/api/server_push"
	"bdsw-im-ws/internal/dubbo/client"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"bdsw-im-ws/internal/redis"
	"bdsw-im-ws/pkg/utils"

	"github.com/gorilla/websocket"
)

type GatewayService struct {
	instanceID  string
	redisClient *redis.Client
	dubboClient *client.Client

	// 连接管理
	userConnections *UserConnectionManager
	websocketServer *WebSocketServer

	// 流式连接
	serverPushStream server_push.ServerPushStreamClient
	streamMutex      sync.RWMutex

	// Token 缓存
	tokenCache *TokenCache

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
			Port:            "8080",
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}),
		tokenCache:   NewTokenCache(5*time.Minute, 10*time.Minute),
		shutdownChan: make(chan struct{}),
	}
}

func (s *GatewayService) SetDubboClient(client *client.Client) {
	s.dubboClient = client
}

func (s *GatewayService) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return fmt.Errorf("service already started")
	}

	// 1. 启动 WebSocket 服务器
	go s.websocketServer.Start(s.handleWebSocketConnection)

	// 2. 连接到业务服务的推送流
	go s.connectToServerPushStream()

	// 3. 启动健康检查
	go s.startHealthCheck()

	// 4. 启动统计报告
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

	// 1. 关闭推送流
	if s.serverPushStream != nil {
		s.serverPushStream.CloseSend()
	}

	// 2. 关闭所有 WebSocket 连接
	s.websocketServer.Shutdown()

	// 3. 清理 Redis 注册信息
	s.cleanupRedisRegistrations()

	s.started = false
	log.Println("IMA Gateway shutdown completed")
	return nil
}

// 确保不复制 websocket.Conn
func (s *GatewayService) handleWebSocketConnection(conn *websocket.Conn, userID string, token string, deviceID string) {
	log.Printf("WebSocket connection attempt for user: %s, device: %s", userID, deviceID)

	// 1. Token 验证
	authResult, err := s.validateToken(userID, token, deviceID)
	if err != nil || !authResult.Valid {
		log.Printf("Token validation failed for user %s: %v", userID, err)

		// 发送验证失败消息后关闭连接
		s.sendAuthResult(conn, false, "Token validation failed")
		conn.Close()
		return
	}

	// 2. 检查是否多端登录
	if s.shouldKickExistingConnection(userID, deviceID) {
		s.kickExistingConnection(userID, deviceID, "multi_login")
	}

	// 3. 注册用户连接 - 传递指针
	s.userConnections.Register(userID, deviceID, conn)

	// 4. 注册到 Redis
	s.registerUserConnection(userID, deviceID, authResult)

	// 5. 通知业务服务用户上线
	s.notifyUserStatusChange(userID, deviceID, "online")

	// 6. 发送验证成功消息
	s.sendAuthResult(conn, true, "Authentication successful")

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

func (s *GatewayService) validateToken(userID, token, deviceID string) (*muc.TokenValidateResponse, error) {
	// 1. 先检查缓存
	if cachedResult := s.tokenCache.Get(userID, token); cachedResult != nil {
		if cachedResult.Valid && cachedResult.ExpireTime > time.Now().Unix() {
			log.Printf("Token cache hit for user: %s", userID)
			return cachedResult, nil
		}
	}

	// 2. 调用 MUC 服务验证
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	authResult, err := s.dubboClient.MUCService.ValidateToken(ctx, &muc.TokenValidateRequest{
		UserId:   userID,
		Token:    token,
		DeviceId: deviceID,
	})
	if err != nil {
		log.Printf("Token validation RPC error for user %s: %v", userID, err)
		return nil, err
	}

	// 3. 缓存验证结果
	if authResult.Valid {
		s.tokenCache.Set(userID, token, authResult)
	}

	return authResult, nil
}

func (s *GatewayService) shouldKickExistingConnection(userID, deviceID string) bool {
	// 根据业务策略决定是否踢掉现有连接
	existingDevices := s.userConnections.GetUserDevices(userID)

	if len(existingDevices) == 0 {
		return false
	}

	// 示例策略：同设备允许，不同设备不允许
	for _, existingDevice := range existingDevices {
		if existingDevice != deviceID {
			return true
		}
	}

	return false
}

func (s *GatewayService) kickExistingConnection(userID, deviceID, reason string) {
	if conn := s.userConnections.Get(userID, deviceID); conn != nil {
		// 发送被踢下线的消息
		kickMsg := &common.ServerMessage{
			MessageId: utils.GenerateMessageID(),
			Type:      "auth",
			Action:    "kicked",
			Data:      []byte(fmt.Sprintf(`{"reason": "%s"}`, reason)),
			Timestamp: time.Now().Unix(),
			Status:    "info",
		}

		conn.WriteJSON(kickMsg)
		conn.Close()

		s.userConnections.Unregister(userID, deviceID)
		s.unregisterUserConnection(userID, deviceID)

		log.Printf("Kicked user %s (device: %s) for reason: %s", userID, deviceID, reason)
	}
}

func (s *GatewayService) sendAuthResult(conn *websocket.Conn, success bool, message string) {
	authResponse := &common.ServerMessage{
		MessageId: utils.GenerateMessageID(),
		Type:      "auth",
		Action:    "login_result",
		Timestamp: time.Now().Unix(),
		Status:    "success",
	}

	if !success {
		authResponse.Status = "error"
		authResponse.ErrorMsg = message
	} else {
		authResponse.Data = []byte(fmt.Sprintf(`{"message": "%s"}`, message))
	}

	conn.WriteJSON(authResponse)
}

func (s *GatewayService) handleClientMessage(userID, deviceID string, rawMessage []byte) {
	startTime := time.Now()

	// 解析基础消息格式
	var clientMsg common.ClientMessage
	if err := json.Unmarshal(rawMessage, &clientMsg); err != nil {
		log.Printf("Failed to parse message from user %s: %v", userID, err)
		s.sendErrorToUser(userID, deviceID, "Invalid message format")
		return
	}

	// 设置用户ID和设备ID
	clientMsg.UserId = userID
	clientMsg.DeviceId = deviceID
	clientMsg.Timestamp = time.Now().Unix()

	// 在消息数据中添加设备信息
	if len(clientMsg.Data) == 0 {
		clientMsg.Data = []byte("{}")
	}

	var dataMap map[string]interface{}
	if err := json.Unmarshal(clientMsg.Data, &dataMap); err == nil {
		dataMap["device_id"] = deviceID
		if enhancedData, err := json.Marshal(dataMap); err == nil {
			clientMsg.Data = enhancedData
		}
	}

	// 直接转发到业务服务
	serverMsg, err := s.dubboClient.BusinessMessageService.HandleClientMessage(
		context.Background(),
		&clientMsg,
	)
	if err != nil {
		log.Printf("Failed to process message for user %s: %v", userID, err)
		s.sendErrorToUser(userID, deviceID, "Service temporarily unavailable")
		return
	}

	// 如果有立即响应，直接返回给用户
	if serverMsg != nil {
		if err := s.sendServerMessageToUser(userID, deviceID, serverMsg); err != nil {
			log.Printf("Failed to send response to user %s: %v", userID, err)
		}
	}

	// 记录处理时间
	duration := time.Since(startTime)
	if duration > 100*time.Millisecond {
		log.Printf("Slow message processing for user %s: %v", userID, duration)
	}
}

func (s *GatewayService) connectToServerPushStream() {
	for {
		select {
		case <-s.shutdownChan:
			return
		default:
			if s.dubboClient == nil {
				time.Sleep(2 * time.Second)
				continue
			}

			stream, err := s.dubboClient.ServerPushService.RegisterPushStream(context.Background())
			if err != nil {
				log.Printf("Failed to connect to server push stream: %v. Retrying in 5s...", err)
				time.Sleep(5 * time.Second)
				continue
			}

			s.streamMutex.Lock()
			s.serverPushStream = stream
			s.streamMutex.Unlock()

			log.Println("Connected to server push stream successfully")

			// 发送初始心跳
			s.sendHeartbeat(stream)

			// 处理服务端推送消息
			for {
				select {
				case <-s.shutdownChan:
					return
				default:
					pushRequest, err := stream.Recv()
					if err != nil {
						log.Printf("Server push stream error: %v", err)
						break
					}
					s.handleServerPush(pushRequest)
				}
			}

			s.streamMutex.Lock()
			s.serverPushStream = nil
			s.streamMutex.Unlock()

			time.Sleep(2 * time.Second)
		}
	}
}

func (s *GatewayService) sendHeartbeat(stream server_push.ServerPushStreamClient) {
	heartbeat := &server_push.Heartbeat{
		ImaInstance: s.instanceID,
		Timestamp:   time.Now().Unix(),
		OnlineCount: int32(s.userConnections.GetOnlineCount()),
	}

	if err := stream.Send(heartbeat); err != nil {
		log.Printf("Failed to send heartbeat: %v", err)
	}
}

func (s *GatewayService) handleServerPush(pushRequest *ima_gateway.PushRequest) {
	log.Printf("Received server push for user: %s, device: %s", pushRequest.UserId, pushRequest.DeviceId)

	if pushRequest.UserId == "" {
		// 广播消息
		s.broadcastMessage(pushRequest.Message)
	} else if pushRequest.DeviceId == "" {
		// 推送给用户的所有设备
		s.userConnections.BroadcastToUser(pushRequest.UserId, func(deviceID string, conn *websocket.Conn) {
			if err := conn.WriteJSON(pushRequest.Message); err != nil {
				log.Printf("Failed to push to user %s (device: %s): %v", pushRequest.UserId, deviceID, err)
			}
		})
	} else {
		// 单设备推送
		s.sendServerMessageToUser(pushRequest.UserId, pushRequest.DeviceId, pushRequest.Message)
	}
}

// 确保所有 WebSocket 操作都使用指针
func (s *GatewayService) sendServerMessageToUser(userID, deviceID string, serverMsg *common.ServerMessage) error {
	conn := s.userConnections.Get(userID, deviceID)
	if conn == nil {
		return fmt.Errorf("user %s (device: %s) not connected", userID, deviceID)
	}

	// 使用指针操作，不复制连接
	return conn.WriteJSON(serverMsg)
}

func (s *GatewayService) broadcastMessage(serverMsg *common.ServerMessage) {
	s.userConnections.Broadcast(func(userID, deviceID string, conn *websocket.Conn) {
		if err := conn.WriteJSON(serverMsg); err != nil {
			log.Printf("Failed to broadcast to user %s (device: %s): %v", userID, deviceID, err)
		}
	})
}

func (s *GatewayService) sendErrorToUser(userID, deviceID, errorMsg string) {
	conn := s.userConnections.Get(userID, deviceID)
	if conn != nil {
		errorResponse := &common.ServerMessage{
			MessageId: utils.GenerateMessageID(),
			Type:      "error",
			Status:    "error",
			ErrorMsg:  errorMsg,
			Timestamp: time.Now().Unix(),
		}
		conn.WriteJSON(errorResponse)
	}
}

func (s *GatewayService) notifyUserStatusChange(userID, deviceID, status string) {
	// 这里可以扩展为向业务服务发送状态变更通知
	log.Printf("User %s (device: %s) status changed to: %s", userID, deviceID, status)
}

func (s *GatewayService) registerUserConnection(userID, deviceID string, authResult *muc.TokenValidateResponse) {
	connInfo := map[string]interface{}{
		"user_id":      userID,
		"device_id":    deviceID,
		"ima_instance": s.instanceID,
		"login_time":   time.Now().Unix(),
		"last_active":  time.Now().Unix(),
		"user_info":    authResult.UserInfo,
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

func (s *GatewayService) startHealthCheck() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.shutdownChan:
			return
		case <-ticker.C:
			// 发送心跳到推送流
			s.streamMutex.RLock()
			if s.serverPushStream != nil {
				s.sendHeartbeat(s.serverPushStream)
			}
			s.streamMutex.RUnlock()

			// 更新实例状态
			s.updateInstanceStatus()
		}
	}
}

func (s *GatewayService) updateInstanceStatus() {
	status := map[string]interface{}{
		"instance_id":  s.instanceID,
		"online_users": s.userConnections.GetOnlineCount(),
		"last_update":  time.Now().Unix(),
		"status":       "healthy",
	}

	if err := s.redisClient.SetInstanceStatus(s.instanceID, status); err != nil {
		log.Printf("Failed to update instance status: %v", err)
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

// PushToUser 实现单用户推送
func (s *GatewayService) PushToUser(ctx context.Context, req *ima_gateway.PushRequest) (*common.BaseResponse, error) {
	if err := s.sendServerMessageToUser(req.UserId, req.DeviceId, req.Message); err != nil {
		return &common.BaseResponse{
			Success:   false,
			Code:      "PUSH_FAILED",
			Message:   err.Error(),
			Timestamp: time.Now().Unix(),
		}, nil
	}

	return &common.BaseResponse{
		Success:   true,
		Timestamp: time.Now().Unix(),
	}, nil
}

// PushToUsers 实现批量推送
func (s *GatewayService) PushToUsers(ctx context.Context, req *ima_gateway.BatchPushRequest) (*common.BaseResponse, error) {
	successCount := 0
	for _, push := range req.Pushes {
		if err := s.sendServerMessageToUser(push.UserId, push.DeviceId, push.Message); err != nil {
			log.Printf("Failed to push to user %s: %v", push.UserId, err)
		} else {
			successCount++
		}
	}

	return &common.BaseResponse{
		Success:   true,
		Message:   fmt.Sprintf("Successfully pushed to %d users", successCount),
		Timestamp: time.Now().Unix(),
	}, nil
}

// Broadcast 实现广播消息
func (s *GatewayService) Broadcast(ctx context.Context, req *ima_gateway.BroadcastMessage) (*common.BaseResponse, error) {
	s.broadcastMessage(req.Message)

	return &common.BaseResponse{
		Success:   true,
		Timestamp: time.Now().Unix(),
	}, nil
}

// KickUser 实现踢用户下线
func (s *GatewayService) KickUser(ctx context.Context, req *ima_gateway.KickUserRequest) (*common.BaseResponse, error) {
	s.kickExistingConnection(req.UserId, req.DeviceId, req.Reason)

	return &common.BaseResponse{
		Success:   true,
		Timestamp: time.Now().Unix(),
	}, nil
}
