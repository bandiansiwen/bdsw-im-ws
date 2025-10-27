package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bdsw-im-ws/internal/config"
	"bdsw-im-ws/pkg/wsmanager"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWebSocketUpgrader 测试WebSocket升级器
func TestWebSocketUpgrader(t *testing.T) {
	cfg := &config.Config{
		Env: "test",
		WebSocket: struct {
			ReadBufferSize    int   `mapstructure:"WS_READ_BUFFER_SIZE"`
			WriteBufferSize   int   `mapstructure:"WS_WRITE_BUFFER_SIZE"`
			EnableCompression bool  `mapstructure:"WS_ENABLE_COMPRESSION"`
			HandshakeTimeout  int   `mapstructure:"WS_HANDSHAKE_TIMEOUT"`
			MaxMessageSize    int64 `mapstructure:"WS_MAX_MESSAGE_SIZE"`
			PingInterval      int   `mapstructure:"WS_PING_INTERVAL"`
			PongWait          int   `mapstructure:"WS_PONG_WAIT"`
			WriteWait         int   `mapstructure:"WS_WRITE_WAIT"`
		}{
			ReadBufferSize:    1024,
			WriteBufferSize:   1024,
			EnableCompression: true,
			HandshakeTimeout:  10,
			MaxMessageSize:    512,
			PingInterval:      30,
			PongWait:          60,
			WriteWait:         10,
		},
	}

	manager := wsmanager.NewManager()
	handler := NewWSHandler(manager, cfg)

	_ = handler.GetUpgrader()

	// 测试生产环境的域名检查
	cfg.Env = "production"
	upgraderProd := handler.GetUpgrader()

	req, _ := http.NewRequest("GET", "/ws", nil)
	req.Header.Set("Origin", "https://malicious.com")

	// 恶意域名应该被拒绝
	assert.False(t, upgraderProd.CheckOrigin(req))

	// 测试开发环境应该允许所有域名
	cfg.Env = "development"
	upgraderDev := handler.GetUpgrader()
	assert.True(t, upgraderDev.CheckOrigin(req))
}

// TestWebSocketConnection 测试WebSocket连接处理
func TestWebSocketConnection(t *testing.T) {
	// 设置Gin测试模式
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Env: "test",
		WebSocket: struct {
			ReadBufferSize    int   `mapstructure:"WS_READ_BUFFER_SIZE"`
			WriteBufferSize   int   `mapstructure:"WS_WRITE_BUFFER_SIZE"`
			EnableCompression bool  `mapstructure:"WS_ENABLE_COMPRESSION"`
			HandshakeTimeout  int   `mapstructure:"WS_HANDSHAKE_TIMEOUT"`
			MaxMessageSize    int64 `mapstructure:"WS_MAX_MESSAGE_SIZE"`
			PingInterval      int   `mapstructure:"WS_PING_INTERVAL"`
			PongWait          int   `mapstructure:"WS_PONG_WAIT"`
			WriteWait         int   `mapstructure:"WS_WRITE_WAIT"`
		}{
			ReadBufferSize:    1024,
			WriteBufferSize:   1024,
			EnableCompression: true,
			HandshakeTimeout:  10,
			MaxMessageSize:    512,
			PingInterval:      30,
			PongWait:          60,
			WriteWait:         10,
		},
	}

	manager := wsmanager.NewManager()
	handler := NewWSHandler(manager, cfg)

	// 创建测试服务器
	router := gin.New()
	router.Use(func(c *gin.Context) {
		// 模拟认证中间件设置用户ID
		c.Set("userID", "test-user-123")
		c.Next()
	})
	router.GET("/ws", handler.HandleConnection)

	server := httptest.NewServer(router)
	defer server.Close()

	// 测试WebSocket连接
	url := "ws" + server.URL[4:] + "/ws"
	conn, resp, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	defer conn.Close()

	assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)

	// 等待连接被管理器注册
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 1, manager.GetClientCount())

	// 测试消息发送和接收
	testMessage := `{"type": "test", "data": "hello"}`
	err = conn.WriteMessage(websocket.TextMessage, []byte(testMessage))
	require.NoError(t, err)

	// 读取响应（回声测试）
	_, response, err := conn.ReadMessage()
	require.NoError(t, err)

	assert.Contains(t, string(response), "Echo: "+testMessage)

	// 关闭连接
	conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	time.Sleep(100 * time.Millisecond)

	// 连接应该被注销
	assert.Equal(t, 0, manager.GetClientCount())
}

// TestWebSocketHandlerAPIs 测试管理API
func TestWebSocketHandlerAPIs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{Env: "test"}
	manager := wsmanager.NewManager()
	handler := NewWSHandler(manager, cfg)

	router := gin.New()
	router.GET("/api/stats", handler.GetStats)
	router.POST("/api/broadcast", handler.BroadcastMessage)

	// 测试获取统计信息
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/stats", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "success", response["status"])
	assert.Contains(t, response, "data")
}

// TestBroadcastAPI 测试广播API
func TestBroadcastAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{Env: "test"}
	manager := wsmanager.NewManager()
	handler := NewWSHandler(manager, cfg)

	router := gin.New()
	router.POST("/api/broadcast", handler.BroadcastMessage)

	// 测试有效的广播请求
	broadcastData := map[string]string{
		"message": "test broadcast",
	}
	jsonData, _ := json.Marshal(broadcastData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/broadcast", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "success", response["status"])
	assert.Equal(t, "Message broadcasted", response["message"])

	// 测试无效的请求（缺少message字段）
	invalidData := map[string]string{}
	invalidJson, _ := json.Marshal(invalidData)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/broadcast", bytes.NewBuffer(invalidJson))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestClientIDGeneration 测试客户端ID生成
func TestClientIDGeneration(t *testing.T) {
	// 测试多次生成的ID是否唯一
	ids := make(map[string]bool)

	for i := 0; i < 1000; i++ {
		id := generateClientID()
		assert.Len(t, id, 32) // 16字节 = 32字符十六进制
		assert.False(t, ids[id], "Duplicate ID generated: %s", id)
		ids[id] = true
	}
}

// BenchmarkWebSocketRegistration 性能测试：客户端注册
func BenchmarkWebSocketRegistration(b *testing.B) {
	cfg := &config.Config{Env: "test"}
	manager := wsmanager.NewManager()
	_ = NewWSHandler(manager, cfg)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		mockConn := wsmanager.NewMockWebSocketConn()
		client := wsmanager.NewClient(
			"bench-client-"+string(rune(i)),
			mockConn,
			"bench-user-"+string(rune(i%100)),
		)
		manager.Register(client)
	}
}

// BenchmarkBroadcast 性能测试：广播消息
func BenchmarkBroadcast(b *testing.B) {
	_ = &config.Config{Env: "test"}
	manager := wsmanager.NewManager()

	// 预先注册1000个客户端
	for i := 0; i < 1000; i++ {
		mockConn := wsmanager.NewMockWebSocketConn()
		client := wsmanager.NewClient(
			"bench-client-"+string(rune(i)),
			mockConn,
			"bench-user-"+string(rune(i%100)),
		)
		manager.Register(client)
	}

	time.Sleep(500 * time.Millisecond) // 等待注册完成

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		manager.Broadcast([]byte("benchmark message"))
	}
}
