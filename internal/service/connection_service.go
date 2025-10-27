package service

import (
	"encoding/json"
	"log"
	"time"

	"bdsw-im-ws/pkg/wsmanager"
)

// ConnectionService 连接服务 - 只处理最基础的连接功能
type ConnectionService struct {
	manager *wsmanager.Manager
}

// NewConnectionService 创建连接服务
func NewConnectionService(manager *wsmanager.Manager) *ConnectionService {
	return &ConnectionService{
		manager: manager,
	}
}

// HandleClientMessage 处理客户端消息 - 现在只处理ping
func (s *ConnectionService) HandleClientMessage(client *wsmanager.Client, rawMessage []byte) bool {
	var message map[string]interface{}
	if err := json.Unmarshal(rawMessage, &message); err != nil {
		// 连错误都不处理，全部交给业务服务
		return false
	}

	msgType, _ := message["type"].(string)

	// 只处理ping消息，其他所有消息都转发到业务服务
	if msgType == "ping" {
		s.handlePing(client)
		return true
	}

	return false
}

// handlePing 处理心跳 - 这是连接服务唯一处理的业务
func (s *ConnectionService) handlePing(client *wsmanager.Client) {
	response := map[string]interface{}{
		"type":      "pong",
		"timestamp": time.Now().Unix(),
	}

	s.sendJSON(client, response)
}

// sendJSON 发送JSON响应
func (s *ConnectionService) sendJSON(client *wsmanager.Client, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to marshal response: %v", err)
		return
	}

	select {
	case client.Send <- jsonData:
		// 发送成功
	default:
		log.Printf("Client %s send channel is full", client.ID)
	}
}

// SendToClient 向特定客户端发送消息
func (s *ConnectionService) SendToClient(client *wsmanager.Client, message interface{}) {
	s.sendJSON(client, message)
}
