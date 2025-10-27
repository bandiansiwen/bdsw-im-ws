package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"bdsw-im-ws/internal/config"
	"bdsw-im-ws/internal/service"
	"bdsw-im-ws/pkg/service_factory"
	"bdsw-im-ws/pkg/wsmanager"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 生产环境应该严格检查
	},
}

type WSHandler struct {
	manager           *wsmanager.Manager
	cfg               *config.Config
	connectionService *service.ConnectionService
	serviceFactory    *service_factory.ServiceFactory
}

func NewWSHandler(manager *wsmanager.Manager, cfg *config.Config, serviceFactory *service_factory.ServiceFactory) *WSHandler {
	connectionService := service.NewConnectionService(manager)

	return &WSHandler{
		manager:           manager,
		cfg:               cfg,
		connectionService: connectionService,
		serviceFactory:    serviceFactory,
	}
}

func (h *WSHandler) HandleConnection(c *gin.Context) {
	// 检查认证服务是否可用
	if !h.serviceFactory.HasAuthService() {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authentication service unavailable"})
		return
	}

	// 1. 首先验证token
	token := h.extractToken(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication token required"})
		return
	}

	// 2. 调用用户服务验证token
	authClient := h.serviceFactory.GetAuthClient()
	userInfo, err := authClient.VerifyToken(token)
	if err != nil {
		log.Printf("Token verification failed: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication failed"})
		return
	}

	// 3. 只有验证通过才建立WebSocket连接
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// 4. 创建已认证的客户端
	wrappedConn := wsmanager.NewRealWebSocketConn(conn)
	client := wsmanager.NewClient(
		generateClientID(),
		wrappedConn,
		userInfo.UserID,
	)

	client.IP = c.ClientIP()
	client.Platform = c.GetHeader("User-Agent")
	client.SetMetadata("user_info", userInfo)
	client.SetMetadata("authenticated_at", time.Now())

	// 5. 注册到管理器
	h.manager.Register(client)

	// 6. 通知业务服务新连接建立
	if h.serviceFactory.HasBusinessService() {
		businessClient := h.serviceFactory.GetBusinessClient()
		go func() {
			if err := businessClient.NotifyConnectionEstablished(client.ID, client.IP, client.Platform); err != nil {
				log.Printf("Failed to notify business service of new connection: %v", err)
			}
		}()
	}

	log.Printf("Authenticated WebSocket connection: %s, User: %s (%s)",
		client.ID, userInfo.UserID, userInfo.Username)

	go h.writePump(client)
	go h.readPump(client)
}

// extractToken 支持多种token获取方式
func (h *WSHandler) extractToken(c *gin.Context) string {
	// 1. 从Query参数获取
	if token := c.Query("token"); token != "" {
		return token
	}

	// 2. 从Header获取
	authHeader := c.GetHeader("Authorization")
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}

	// 3. 从Cookie获取
	if cookie, err := c.Cookie("auth_token"); err == nil {
		return cookie
	}

	return ""
}

func (h *WSHandler) readPump(client *wsmanager.Client) {
	defer func() {
		h.manager.Unregister(client)

		// 通知业务服务连接断开
		if h.serviceFactory.HasBusinessService() {
			businessClient := h.serviceFactory.GetBusinessClient()
			go func() {
				if err := businessClient.NotifyConnectionClosed(client.ID, client.UserID); err != nil {
					log.Printf("Failed to notify business service of connection close: %v", err)
				}
			}()
		}

		log.Printf("WebSocket connection closed: %s, User: %s", client.ID, client.UserID)
	}()

	// 设置读取参数
	client.Conn.SetReadLimit(h.cfg.WebSocket.MaxMessageSize)
	client.Conn.SetReadDeadline(time.Now().Add(time.Duration(h.cfg.WebSocket.PongWait) * time.Second))
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(time.Duration(h.cfg.WebSocket.PongWait) * time.Second))
		return nil
	})

	for {
		messageType, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Read error: %v", err)
			}
			break
		}

		// 只处理文本消息
		if messageType == websocket.TextMessage {
			h.handleMessage(client, message)
		}
	}
}

func (h *WSHandler) handleMessage(client *wsmanager.Client, message []byte) {
	// 先让连接服务处理最基础的消息（目前只有ping）
	messageHandled := h.connectionService.HandleClientMessage(client, message)

	// 如果有业务服务，并且消息没有被连接服务处理，则转发到业务服务
	if h.serviceFactory.HasBusinessService() && !messageHandled {
		go h.forwardToBusinessService(client, message)
	}
}

// forwardToBusinessService 转发所有消息到业务服务
func (h *WSHandler) forwardToBusinessService(client *wsmanager.Client, message []byte) {
	var msgData map[string]interface{}
	if err := json.Unmarshal(message, &msgData); err != nil {
		log.Printf("Failed to unmarshal message for business service: %v", err)
		return
	}

	// 记录转发的消息（用于监控）
	msgType, _ := msgData["type"].(string)
	log.Printf("Forwarding message to business service - Type: %s, Client: %s",
		msgType, client.ID)

	// 转发到业务服务
	businessClient := h.serviceFactory.GetBusinessClient()
	response, err := businessClient.ForwardMessage(client.ID, client.UserID, msgData, client.IP)
	if err != nil {
		log.Printf("Failed to forward message to business service: %v", err)
		return
	}

	// 处理业务服务的响应
	h.handleBusinessResponse(client, response)
}

// handleBusinessResponse 处理业务服务的响应
func (h *WSHandler) handleBusinessResponse(client *wsmanager.Client, response *service.MessageResponse) {
	if response == nil {
		return
	}

	// 处理需要发送给客户端的响应
	if response.SendToClient {
		h.connectionService.SendToClient(client, response.Data)
	}

	// 处理需要发送给其他客户端的消息
	for _, targetMsg := range response.TargetMessages {
		// 这里可以实现根据clientID或userID发送消息给其他客户端
		// 需要扩展manager来支持根据ID查找客户端
		log.Printf("Target message to be sent - ClientID: %s, UserID: %s",
			targetMsg.ClientID, targetMsg.UserID)
	}
}

func (h *WSHandler) writePump(client *wsmanager.Client) {
	ticker := time.NewTicker(time.Duration(h.cfg.WebSocket.PingInterval) * time.Second)
	defer func() {
		ticker.Stop()
		client.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(time.Duration(h.cfg.WebSocket.WriteWait) * time.Second))

			if !ok {
				// 通道关闭，发送关闭消息
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// 写入消息
			if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Write message error: %v", err)
				return
			}

		case <-ticker.C:
			// 发送心跳
			client.Conn.SetWriteDeadline(time.Now().Add(time.Duration(h.cfg.WebSocket.WriteWait) * time.Second))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Ping error: %v", err)
				return
			}
		}
	}
}

// GetStats 获取统计信息
func (h *WSHandler) GetStats(c *gin.Context) {
	stats := h.manager.GetStats()
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   stats,
	})
}

// GetOnlineUsers 获取在线用户列表
func (h *WSHandler) GetOnlineUsers(c *gin.Context) {
	onlineUsers := h.manager.GetOnlineUsers()

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"online_users": onlineUsers,
			"total":        len(onlineUsers),
		},
	})
}

// BroadcastMessage 广播消息（管理接口）
func (h *WSHandler) BroadcastMessage(c *gin.Context) {
	var request struct {
		Message string `json:"message" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	broadcastMsg := map[string]interface{}{
		"type":      "broadcast",
		"message":   request.Message,
		"from":      "system",
		"timestamp": time.Now().Unix(),
	}

	jsonData, err := json.Marshal(broadcastMsg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to marshal message"})
		return
	}

	h.manager.Broadcast(jsonData)

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Message broadcasted",
	})
}

func generateClientID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
