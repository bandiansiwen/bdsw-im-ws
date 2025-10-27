package wsmanager

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestManager_RegisterClient(t *testing.T) {
	manager := NewManager()

	client := &Client{
		ID:     "test-client-1",
		Send:   make(chan []byte, 256),
		UserID: "user-123",
	}

	manager.Register(client)
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 1, manager.GetClientCount())
	assert.Equal(t, 1, manager.GetUserClientCount("user-123"))

	stats := manager.GetStats()
	assert.Equal(t, int64(1), stats["current_connections"])
	assert.Equal(t, int64(1), stats["total_connections"])
}

func TestManager_UnregisterClient(t *testing.T) {
	manager := NewManager()

	client := &Client{
		ID:     "test-client-1",
		Send:   make(chan []byte, 256),
		UserID: "user-123",
	}

	manager.Register(client)
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 1, manager.GetClientCount())

	manager.Unregister(client)
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 0, manager.GetClientCount())
	assert.Equal(t, 0, manager.GetUserClientCount("user-123"))
}

func TestManager_Broadcast(t *testing.T) {
	manager := NewManager()

	// 创建多个客户端
	client1 := &Client{
		ID:     "client-1",
		Send:   make(chan []byte, 256),
		UserID: "user-1",
	}

	client2 := &Client{
		ID:     "client-2",
		Send:   make(chan []byte, 256),
		UserID: "user-2",
	}

	manager.Register(client1)
	manager.Register(client2)
	time.Sleep(100 * time.Millisecond)

	// 广播消息
	testMessage := []byte("broadcast message")
	manager.Broadcast(testMessage)
	time.Sleep(100 * time.Millisecond)

	// 验证消息被发送到客户端的通道
	select {
	case msg1 := <-client1.Send:
		assert.Equal(t, testMessage, msg1)
	default:
		t.Error("Client 1 did not receive broadcast message")
	}

	select {
	case msg2 := <-client2.Send:
		assert.Equal(t, testMessage, msg2)
	default:
		t.Error("Client 2 did not receive broadcast message")
	}
}

func TestManager_SendToUser(t *testing.T) {
	manager := NewManager()

	client1 := &Client{
		ID:     "client-1",
		Send:   make(chan []byte, 256),
		UserID: "user-123",
	}

	client2 := &Client{
		ID:     "client-2",
		Send:   make(chan []byte, 256),
		UserID: "user-123", // 同一用户
	}

	client3 := &Client{
		ID:     "client-3",
		Send:   make(chan []byte, 256),
		UserID: "user-456", // 不同用户
	}

	manager.Register(client1)
	manager.Register(client2)
	manager.Register(client3)
	time.Sleep(100 * time.Millisecond)

	// 发送用户特定消息
	testMessage := []byte("user message")
	manager.SendToUser("user-123", testMessage)
	time.Sleep(100 * time.Millisecond)

	// 检查只有 user-123 的客户端收到消息
	select {
	case msg1 := <-client1.Send:
		assert.Equal(t, testMessage, msg1)
	default:
		t.Error("Client 1 did not receive user message")
	}

	select {
	case msg2 := <-client2.Send:
		assert.Equal(t, testMessage, msg2)
	default:
		t.Error("Client 2 did not receive user message")
	}

	// client3 不应该收到消息
	select {
	case <-client3.Send:
		t.Error("Client 3 should not receive user message")
	default:
		// 正确，没有收到消息
	}
}

func TestManager_ConcurrentOperations(t *testing.T) {
	manager := NewManager()

	var wg sync.WaitGroup
	clientsCount := 100

	// 并发注册客户端
	for i := 0; i < clientsCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			client := &Client{
				ID:     "client-" + string(rune(index)),
				Send:   make(chan []byte, 256),
				UserID: "user-" + string(rune(index%10)),
			}
			manager.Register(client)
		}(i)
	}

	wg.Wait()
	time.Sleep(500 * time.Millisecond)

	assert.Equal(t, clientsCount, manager.GetClientCount())

	stats := manager.GetStats()
	assert.Equal(t, int64(clientsCount), stats["current_connections"])
	assert.Equal(t, int64(clientsCount), stats["total_connections"])
}

func TestManager_ClientMetadata(t *testing.T) {
	manager := NewManager()

	userInfo := &UserInfo{
		UserID:   "user-123",
		Username: "testuser",
		Email:    "test@example.com",
		Roles:    []string{"user"},
	}

	client := &Client{
		ID:     "test-client",
		Send:   make(chan []byte, 256),
		UserID: "user-123",
	}

	client.SetMetadata("user_info", userInfo)
	client.SetMetadata("authenticated_at", time.Now())

	manager.Register(client)
	time.Sleep(100 * time.Millisecond)

	// 测试获取用户信息
	retrievedUserInfo, exists := manager.GetUserInfo("user-123")
	assert.True(t, exists)
	assert.Equal(t, userInfo.UserID, retrievedUserInfo.UserID)
	assert.Equal(t, userInfo.Username, retrievedUserInfo.Username)

	// 测试获取在线用户
	onlineUsers := manager.GetOnlineUsers()
	assert.Len(t, onlineUsers, 1)
	assert.Equal(t, "user-123", onlineUsers[0].UserID)
}

func TestManager_ClientChannelFull(t *testing.T) {
	manager := NewManager()

	// 创建发送通道很小的客户端
	client := &Client{
		ID:     "test-client",
		Send:   make(chan []byte, 1), // 很小的缓冲区
		UserID: "user-123",
	}

	manager.Register(client)
	time.Sleep(100 * time.Millisecond)

	// 发送多条消息使通道满
	for i := 0; i < 3; i++ {
		manager.Broadcast([]byte("message"))
	}

	time.Sleep(100 * time.Millisecond)

	// 客户端应该因为通道满而被注销
	assert.Equal(t, 0, manager.GetClientCount())
}

func BenchmarkManager_RegisterClients(b *testing.B) {
	manager := NewManager()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client := &Client{
			ID:     "bench-client-" + string(rune(i)),
			Send:   make(chan []byte, 256),
			UserID: "bench-user-" + string(rune(i%100)),
		}
		manager.Register(client)
	}
}

func BenchmarkManager_Broadcast(b *testing.B) {
	manager := NewManager()

	// 预先注册1000个客户端
	for i := 0; i < 1000; i++ {
		client := &Client{
			ID:     "bench-client-" + string(rune(i)),
			Send:   make(chan []byte, 256),
			UserID: "bench-user-" + string(rune(i%100)),
		}
		manager.Register(client)
	}

	time.Sleep(500 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.Broadcast([]byte("benchmark message"))
	}
}
