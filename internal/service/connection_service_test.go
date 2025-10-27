package service

import (
	"testing"
	"time"

	"bdsw-im-ws/pkg/wsmanager"

	"github.com/stretchr/testify/assert"
)

func TestConnectionService_HandlePingMessage(t *testing.T) {
	manager := wsmanager.NewManager()
	service := NewConnectionService(manager)

	client := &wsmanager.Client{
		ID:     "test-client",
		Send:   make(chan []byte, 256),
		UserID: "user-123",
	}

	// 测试ping消息
	pingMessage := []byte(`{"type": "ping"}`)
	handled := service.HandleClientMessage(client, pingMessage)

	assert.True(t, handled)

	// 检查是否收到pong响应
	select {
	case response := <-client.Send:
		assert.Contains(t, string(response), "pong")
	case <-time.After(100 * time.Millisecond):
		t.Error("Did not receive pong response")
	}
}

func TestConnectionService_HandleNonPingMessage(t *testing.T) {
	manager := wsmanager.NewManager()
	service := NewConnectionService(manager)

	client := &wsmanager.Client{
		ID:     "test-client",
		Send:   make(chan []byte, 256),
		UserID: "user-123",
	}

	// 测试非ping消息
	otherMessage := []byte(`{"type": "chat", "content": "hello"}`)
	handled := service.HandleClientMessage(client, otherMessage)

	assert.False(t, handled)

	// 检查没有发送响应
	select {
	case <-client.Send:
		t.Error("Should not send response for non-ping messages")
	default:
		// 正确，没有发送消息
	}
}

func TestConnectionService_InvalidJSON(t *testing.T) {
	manager := wsmanager.NewManager()
	service := NewConnectionService(manager)

	client := &wsmanager.Client{
		ID:     "test-client",
		Send:   make(chan []byte, 256),
		UserID: "user-123",
	}

	// 测试无效JSON
	invalidJSON := []byte(`invalid json`)
	handled := service.HandleClientMessage(client, invalidJSON)

	assert.False(t, handled)
}

func TestConnectionService_SendToClient(t *testing.T) {
	manager := wsmanager.NewManager()
	service := NewConnectionService(manager)

	client := &wsmanager.Client{
		ID:     "test-client",
		Send:   make(chan []byte, 256),
		UserID: "user-123",
	}

	testMessage := map[string]interface{}{
		"type":    "test",
		"content": "hello",
	}

	service.SendToClient(client, testMessage)

	select {
	case response := <-client.Send:
		assert.Contains(t, string(response), "test")
		assert.Contains(t, string(response), "hello")
	case <-time.After(100 * time.Millisecond):
		t.Error("Did not receive message")
	}
}
