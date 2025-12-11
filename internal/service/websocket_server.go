package service

import (
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

type WebSocketConfig struct {
	Port            string
	ReadBufferSize  int
	WriteBufferSize int
}

type WebSocketServer struct {
	config   *WebSocketConfig
	upgrader websocket.Upgrader
}

func NewWebSocketServer(config *WebSocketConfig) *WebSocketServer {
	return &WebSocketServer{
		config: config,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  config.ReadBufferSize,
			WriteBufferSize: config.WriteBufferSize,
			CheckOrigin: func(r *http.Request) bool {
				// 生产环境应该验证来源
				origin := r.Header.Get("Origin")
				if origin != "" {
					return strings.Contains(origin, "yourdomain.com")
				}
				return true
			},
		},
	}
}

// 确保传递的是连接指针
func (s *WebSocketServer) Start(connectionHandler func(conn *websocket.Conn, userID string, token string, deviceID string)) {
	http.HandleFunc("/ws", func(rw http.ResponseWriter, r *http.Request) {
		s.handleWebSocket(rw, r, connectionHandler)
	})

	// 健康检查端点
	http.HandleFunc("/health", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		rw.Write([]byte(`{"status": "healthy", "service": "ima-gateway"}`))
	})

	log.Printf("Starting WebSocket server on port %s", s.config.Port)

	if err := http.ListenAndServe(":"+s.config.Port, nil); err != nil {
		log.Fatalf("WebSocket server failed: %v", err)
	}
}

// 传递连接指针而不是值
func (s *WebSocketServer) handleWebSocket(rw http.ResponseWriter, r *http.Request, handler func(conn *websocket.Conn, userID string, token string, deviceID string)) {
	// 从查询参数获取用户ID、token和设备ID
	userID := strings.TrimSpace(r.URL.Query().Get("user_id"))
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	deviceID := strings.TrimSpace(r.URL.Query().Get("device_id"))

	if userID == "" || token == "" {
		http.Error(rw, "Missing user_id or token", http.StatusBadRequest)
		return
	}

	// 如果没有提供设备ID，生成一个默认的
	if deviceID == "" {
		deviceID = "web-" + generateRandomString(8)
	}

	// 升级到 WebSocket 连接
	conn, err := s.upgrader.Upgrade(rw, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// 调用处理函数 - 传递连接指针
	handler(conn, userID, token, deviceID)
}

func generateRandomString(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[i%len(chars)]
	}
	return string(result)
}
