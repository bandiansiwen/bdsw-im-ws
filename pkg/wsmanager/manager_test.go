// pkg/wsmanager/manager_test.go
package wsmanager

import (
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func TestManager_RegisterWithMockConnection(t *testing.T) {
	manager := NewManager()

	// 现在可以这样使用！
	mockConn := NewMockWebSocketConn()

	client := NewClient("test-client-1", mockConn, "user-123")
	manager.Register(client)

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 1, manager.GetClientCount())
	assert.Equal(t, 1, manager.GetUserClientCount("user-123"))
}

func TestManager_BroadcastWithMock(t *testing.T) {
	manager := NewManager()

	// 创建多个模拟连接
	mockConn1 := NewMockWebSocketConn().(*MockWebSocketConn)
	mockConn2 := NewMockWebSocketConn().(*MockWebSocketConn)

	client1 := NewClient("client-1", mockConn1, "user-1")
	client2 := NewClient("client-2", mockConn2, "user-2")

	manager.Register(client1)
	manager.Register(client2)
	time.Sleep(100 * time.Millisecond)

	// 广播消息
	testMessage := []byte("broadcast message")
	manager.Broadcast(testMessage)
	time.Sleep(100 * time.Millisecond)

	// 验证消息被正确发送
	assert.Len(t, mockConn1.GetWriteMessages(), 1)
	assert.Len(t, mockConn2.GetWriteMessages(), 1)
	assert.Equal(t, testMessage, mockConn1.GetWriteMessages()[0])
	assert.Equal(t, testMessage, mockConn2.GetWriteMessages()[0])
}

func TestManager_MessageTypes(t *testing.T) {
	manager := NewManager()
	mockConn := NewMockWebSocketConn().(*MockWebSocketConn)

	client := NewClient("test-client", mockConn, "user-123")
	manager.Register(client)
	time.Sleep(100 * time.Millisecond)

	// 测试不同类型消息
	textMessage := []byte("text message")
	manager.Broadcast(textMessage)
	time.Sleep(100 * time.Millisecond)

	// 验证消息类型
	types := mockConn.GetWriteMessageTypes()
	assert.Len(t, types, 1)
	assert.Equal(t, websocket.TextMessage, types[0])
}

func TestManager_ReadMessages(t *testing.T) {
	manager := NewManager()
	mockConn := NewMockWebSocketConn().(*MockWebSocketConn)

	client := NewClient("test-client", mockConn, "user-123")
	manager.Register(client)

	// 模拟接收消息
	testMessage := []byte("test read message")
	mockConn.AddReadMessage(websocket.TextMessage, testMessage)

	// 这里可以测试 readPump 逻辑
	// 需要配合适当的测试框架
}
