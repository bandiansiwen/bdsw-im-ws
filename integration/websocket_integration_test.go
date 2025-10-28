package integration

import (
	"bdsw-im-ws/internal/model"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"bdsw-im-ws/internal/config"
	"bdsw-im-ws/internal/handler"
	"bdsw-im-ws/internal/service"
	"bdsw-im-ws/pkg/wsmanager"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockServiceFactory 模拟服务工厂
type MockServiceFactory struct {
	authClient     *service.AuthClient
	businessClient *service.BusinessServiceClient
}

func (m *MockServiceFactory) GetAuthClient() *service.AuthClient {
	return m.authClient
}

func (m *MockServiceFactory) GetBusinessClient() *service.BusinessServiceClient {
	return m.businessClient
}

func (m *MockServiceFactory) HasAuthService() bool {
	return m.authClient != nil
}

func (m *MockServiceFactory) HasBusinessService() bool {
	return m.businessClient != nil
}

func (m *MockServiceFactory) Close() {}

// MockAuthClient 模拟认证客户端
type MockAuthClient struct{}

func (m *MockAuthClient) VerifyToken(token string) (*model.UserInfo, error) {
	return &model.UserInfo{
		UserID:   "test-user-123",
		Username: "testuser",
	}, nil
}

// TestWebSocketIntegration 集成测试
func TestWebSocketIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{Env: "test"}
	manager := wsmanager.NewManager()

	// 创建模拟服务工厂
	mockAuthClient := &MockAuthClient{}
	mockFactory := &MockServiceFactory{
		authClient: mockAuthClient,
	}

	handler := handler.NewWSHandler(manager, cfg, mockFactory)

	// 创建测试服务器
	router := gin.New()
	router.GET("/ws", handler.HandleConnection)

	server := httptest.NewServer(router)
	defer server.Close()

	// 连接到WebSocket
	url := "ws" + server.URL[4:] + "/ws?token=test-token"
	conn, resp, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	defer conn.Close()

	assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)

	// 等待连接建立
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 1, manager.GetClientCount())

	// 测试发送ping消息
	err = conn.WriteMessage(websocket.TextMessage, []byte(`{"type": "ping"}`))
	require.NoError(t, err)

	// 读取pong响应
	_, response, err := conn.ReadMessage()
	require.NoError(t, err)
	assert.Contains(t, string(response), "pong")

	// 关闭连接
	conn.Close()
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 0, manager.GetClientCount())
}

// TestWebSocketStress 压力测试
func TestWebSocketStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	cfg := &config.Config{Env: "test"}
	manager := wsmanager.NewManager()

	mockAuthClient := &MockAuthClient{}
	mockFactory := &MockServiceFactory{
		authClient: mockAuthClient,
	}

	handler := handler.NewWSHandler(manager, cfg, mockFactory)

	router := gin.New()
	router.GET("/ws", handler.HandleConnection)

	server := httptest.NewServer(router)
	defer server.Close()

	// 并发连接测试
	const concurrentConnections = 50
	var connected int
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < concurrentConnections; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			url := "ws" + server.URL[4:] + "/ws?token=test-token"
			conn, _, err := websocket.DefaultDialer.Dial(url, nil)
			if err != nil {
				t.Logf("Connection %d failed: %v", id, err)
				return
			}
			defer conn.Close()

			mu.Lock()
			connected++
			mu.Unlock()

			// 保持连接一段时间
			time.Sleep(500 * time.Millisecond)
		}(i)
	}

	wg.Wait()

	mu.Lock()
	t.Logf("Successfully connected: %d/%d", connected, concurrentConnections)
	mu.Unlock()

	// 等待所有连接处理完成
	time.Sleep(100 * time.Millisecond)

	// 验证连接数
	assert.Equal(t, 0, manager.GetClientCount()) // 所有连接应该都已关闭
}
