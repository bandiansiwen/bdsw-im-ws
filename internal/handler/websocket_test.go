package handler

import (
	"bdsw-im-ws/internal/model"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bdsw-im-ws/internal/config"
	"bdsw-im-ws/internal/service"
	_ "bdsw-im-ws/pkg/service_factory"
	"bdsw-im-ws/pkg/wsmanager"

	"github.com/gin-gonic/gin"
	_ "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockServiceFactory 用于测试的模拟服务工厂
type MockServiceFactory struct {
	authClient         *service.AuthClient
	businessClient     *service.BusinessServiceClient
	hasAuthService     bool
	hasBusinessService bool
}

func (m *MockServiceFactory) GetAuthClient() *service.AuthClient {
	return m.authClient
}

func (m *MockServiceFactory) GetBusinessClient() *service.BusinessServiceClient {
	return m.businessClient
}

func (m *MockServiceFactory) HasAuthService() bool {
	return m.hasAuthService
}

func (m *MockServiceFactory) HasBusinessService() bool {
	return m.hasBusinessService
}

func (m *MockServiceFactory) Init() error {
	return nil
}

func (m *MockServiceFactory) Close() {}

// MockAuthClient 模拟认证客户端
type MockAuthClient struct {
	shouldFail bool
	userInfo   *model.UserInfo
}

func (m *MockAuthClient) VerifyToken(token string) (*model.UserInfo, error) {
	if m.shouldFail {
		return nil, assert.AnError
	}
	return m.userInfo, nil
}

func TestWebSocketHandler_HandleConnection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{Env: "test"}
	manager := wsmanager.NewManager()

	// 创建模拟服务工厂
	mockAuthClient := &MockAuthClient{
		userInfo: &model.UserInfo{
			UserID:   "test-user-123",
			Username: "testuser",
		},
	}

	mockFactory := &MockServiceFactory{
		authClient:     mockAuthClient,
		hasAuthService: true,
	}

	handler := NewWSHandler(manager, cfg, mockFactory)

	// 创建测试路由
	router := gin.New()
	router.GET("/ws", handler.HandleConnection)

	// 测试没有token的情况
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ws", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// 测试认证失败的情况
	mockAuthClient.shouldFail = true
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/ws?token=invalid", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// 测试认证服务不可用
	mockFactory.hasAuthService = false
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/ws?token=valid", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestWebSocketHandler_ExtractToken(t *testing.T) {
	cfg := &config.Config{Env: "test"}
	manager := wsmanager.NewManager()
	mockFactory := &MockServiceFactory{}
	handler := NewWSHandler(manager, cfg, mockFactory)

	// 测试从Query参数获取token
	req, _ := http.NewRequest("GET", "/ws?token=query-token", nil)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req

	token := handler.extractToken(c)
	assert.Equal(t, "query-token", token)

	// 测试从Header获取token
	req, _ = http.NewRequest("GET", "/ws", nil)
	req.Header.Set("Authorization", "Bearer header-token")
	c.Request = req

	token = handler.extractToken(c)
	assert.Equal(t, "header-token", token)

	// 测试从Cookie获取token
	req, _ = http.NewRequest("GET", "/ws", nil)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: "cookie-token"})
	c.Request = req

	token = handler.extractToken(c)
	assert.Equal(t, "cookie-token", token)

	// 测试没有token的情况
	req, _ = http.NewRequest("GET", "/ws", nil)
	c.Request = req

	token = handler.extractToken(c)
	assert.Equal(t, "", token)
}

func TestWebSocketHandler_APIs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{Env: "test"}
	manager := wsmanager.NewManager()
	mockFactory := &MockServiceFactory{}
	handler := NewWSHandler(manager, cfg, mockFactory)

	// 注册一些测试客户端
	client1 := &wsmanager.Client{
		ID:     "client-1",
		Send:   make(chan []byte, 256),
		UserID: "user-1",
	}
	client1.SetMetadata("user_info", &model.UserInfo{
		UserID:   "user-1",
		Username: "user1",
	})

	client2 := &wsmanager.Client{
		ID:     "client-2",
		Send:   make(chan []byte, 256),
		UserID: "user-2",
	}
	client2.SetMetadata("user_info", &model.UserInfo{
		UserID:   "user-2",
		Username: "user2",
	})

	manager.Register(client1)
	manager.Register(client2)
	time.Sleep(100 * time.Millisecond)

	router := gin.New()
	router.GET("/api/stats", handler.GetStats)
	router.GET("/api/users/online", handler.GetOnlineUsers)
	router.POST("/api/broadcast", handler.BroadcastMessage)

	// 测试获取统计信息
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/stats", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var statsResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &statsResponse)
	require.NoError(t, err)

	assert.Equal(t, "success", statsResponse["status"])
	assert.Contains(t, statsResponse, "data")

	// 测试获取在线用户
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/users/online", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var usersResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &usersResponse)
	require.NoError(t, err)

	assert.Equal(t, "success", usersResponse["status"])

	// 测试广播消息
	broadcastData := map[string]string{
		"message": "test broadcast",
	}
	jsonData, _ := json.Marshal(broadcastData)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/broadcast", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// 清理
	manager.Unregister(client1)
	manager.Unregister(client2)
}

func TestGenerateClientID(t *testing.T) {
	ids := make(map[string]bool)

	// 测试多次生成的ID是否唯一
	for i := 0; i < 100; i++ {
		id := generateClientID()
		assert.Len(t, id, 32) // 16字节 = 32字符十六进制
		assert.False(t, ids[id], "Duplicate ID generated: %s", id)
		ids[id] = true
	}
}

func BenchmarkWebSocketHandler_HandleMessage(b *testing.B) {
	cfg := &config.Config{Env: "test"}
	manager := wsmanager.NewManager()
	mockFactory := &MockServiceFactory{}
	handler := NewWSHandler(manager, cfg, mockFactory)

	client := &wsmanager.Client{
		ID:     "bench-client",
		Send:   make(chan []byte, 256),
		UserID: "bench-user",
	}

	pingMessage := []byte(`{"type": "ping"}`)
	chatMessage := []byte(`{"type": "chat", "content": "hello"}`)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			handler.handleMessage(client, pingMessage)
		} else {
			handler.handleMessage(client, chatMessage)
		}
	}
}
