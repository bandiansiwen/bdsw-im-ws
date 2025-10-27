package handler

import (
	"bdsw-im-ws/pkg/wsmanager"
)

// 测试辅助函数
func createTestMockConn() *wsmanager.MockWebSocketConn {
	mockConn := wsmanager.NewMockWebSocketConn()
	return mockConn.(*wsmanager.MockWebSocketConn)
}

func createTestClient(id, userID string) (*wsmanager.Client, *wsmanager.MockWebSocketConn) {
	mockConn := createTestMockConn()
	client := wsmanager.NewClient(id, mockConn, userID)
	return client, mockConn
}
