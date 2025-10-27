package handler

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"time"

	"bdsw-im-ws/internal/config"
	"bdsw-im-ws/pkg/wsmanager"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type WSHandler struct {
	manager *wsmanager.Manager
	cfg     *config.Config
}

func NewWSHandler(manager *wsmanager.Manager, cfg *config.Config) *WSHandler {
	return &WSHandler{
		manager: manager,
		cfg:     cfg,
	}
}

func (h *WSHandler) GetUpgrader() websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:    h.cfg.WebSocket.ReadBufferSize,
		WriteBufferSize:   h.cfg.WebSocket.WriteBufferSize,
		EnableCompression: h.cfg.WebSocket.EnableCompression,
		CheckOrigin: func(r *http.Request) bool {
			// 生产环境应该严格检查
			if h.cfg.Env == "production" {
				origin := r.Header.Get("Origin")
				// 在这里添加允许的域名
				allowedOrigins := []string{
					"https://yourdomain.com",
				}
				for _, allowed := range allowedOrigins {
					if origin == allowed {
						return true
					}
				}
				return false
			}
			return true
		},
		HandshakeTimeout: time.Duration(h.cfg.WebSocket.HandshakeTimeout) * time.Second,
	}
}

func (h *WSHandler) HandleConnection(c *gin.Context) {
	upgrader := h.GetUpgrader()

	// 升级WebSocket连接
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// 关键改动：使用适配器包装真实连接
	wrappedConn := wsmanager.NewRealWebSocketConn(conn)

	// 获取用户信息（从中间件中设置）
	userID, _ := c.Get("userID")
	userIDStr, _ := userID.(string)

	// 创建客户端
	client := wsmanager.NewClient(
		generateClientID(),
		wrappedConn,
		userIDStr,
	)

	// 设置客户端额外信息
	client.IP = c.ClientIP()
	client.Platform = c.GetHeader("User-Agent")

	// 注册到管理器
	h.manager.Register(client)

	// 启动读写协程
	go h.writePump(client)
	go h.readPump(client)

	log.Printf("New WebSocket connection: %s, User: %s, IP: %s",
		client.ID, client.UserID, client.IP)
}

func (h *WSHandler) writePump(client *wsmanager.Client) {
	ticker := time.NewTicker(time.Duration(h.cfg.WebSocket.PingInterval) * time.Second)
	defer func() {
		ticker.Stop()
		client.Conn.Close()
		h.manager.Unregister(client)
	}()

	for {
		select {
		case message, ok := <-client.Send:
			// 设置写超时
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

func (h *WSHandler) readPump(client *wsmanager.Client) {
	defer func() {
		h.manager.Unregister(client)
		client.Conn.Close()
	}()

	// 设置读取限制
	client.Conn.SetReadLimit(h.cfg.WebSocket.MaxMessageSize)

	// 设置Pong处理器
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(time.Duration(h.cfg.WebSocket.PongWait) * time.Second))
		return nil
	})

	// 设置初始读取超时
	client.Conn.SetReadDeadline(time.Now().Add(time.Duration(h.cfg.WebSocket.PongWait) * time.Second))

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
	log.Printf("Received message from %s: %s", client.ID, string(message))

	// 这里处理业务逻辑
	// 例如：解析JSON，根据消息类型处理

	// 示例：回声测试
	response := []byte("Echo: " + string(message))
	select {
	case client.Send <- response:
		// 消息成功发送到通道
	default:
		log.Printf("Client %s send channel is full", client.ID)
	}
}

func generateClientID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// 提供管理接口
func (h *WSHandler) GetStats(c *gin.Context) {
	stats := h.manager.GetStats()
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   stats,
	})
}

func (h *WSHandler) BroadcastMessage(c *gin.Context) {
	var request struct {
		Message string `json:"message" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.manager.Broadcast([]byte(request.Message))

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Message broadcasted",
	})
}
