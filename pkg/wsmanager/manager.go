package wsmanager

import (
	"log"
	"sync"
	"time"
)

type Manager struct {
	clients     map[string]*Client  // clientID -> Client
	userClients map[string][]string // userID -> []clientID
	broadcast   chan []byte
	register    chan *Client
	unregister  chan *Client
	mu          sync.RWMutex

	// 统计信息
	stats struct {
		totalConnections   int64
		currentConnections int64
		maxConnections     int64
	}
}

func NewManager() *Manager {
	m := &Manager{
		clients:     make(map[string]*Client),
		userClients: make(map[string][]string),
		broadcast:   make(chan []byte, 1024),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
	}

	go m.run()
	go m.monitorStats()

	return m
}

func (m *Manager) run() {
	for {
		select {
		case client := <-m.register:
			m.handleRegister(client)

		case client := <-m.unregister:
			m.handleUnregister(client)

		case message := <-m.broadcast:
			m.handleBroadcast(message)
		}
	}
}

func (m *Manager) handleRegister(client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.clients[client.ID] = client

	// 维护用户到客户端的映射
	if client.UserID != "" {
		m.userClients[client.UserID] = append(m.userClients[client.UserID], client.ID)
	}

	// 更新统计
	m.stats.totalConnections++
	m.stats.currentConnections++
	if m.stats.currentConnections > m.stats.maxConnections {
		m.stats.maxConnections = m.stats.currentConnections
	}

	log.Printf("Client registered: %s, User: %s, Total: %d",
		client.ID, client.UserID, m.stats.currentConnections)
}

func (m *Manager) handleUnregister(client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if c, exists := m.clients[client.ID]; exists {
		close(c.Send)
		delete(m.clients, client.ID)
		m.stats.currentConnections--

		// 清理用户映射
		if client.UserID != "" {
			if clients, ok := m.userClients[client.UserID]; ok {
				for i, id := range clients {
					if id == client.ID {
						m.userClients[client.UserID] = append(clients[:i], clients[i+1:]...)
						break
					}
				}
				if len(m.userClients[client.UserID]) == 0 {
					delete(m.userClients, client.UserID)
				}
			}
		}

		log.Printf("Client unregistered: %s, Remaining: %d", client.ID, m.stats.currentConnections)
	}
}

func (m *Manager) handleBroadcast(message []byte) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, client := range m.clients {
		select {
		case client.Send <- message:
			// 消息成功发送到通道
		default:
			// 通道已满，关闭连接
			go m.Unregister(client)
		}
	}
}

func (m *Manager) Register(client *Client) {
	m.register <- client
}

func (m *Manager) Unregister(client *Client) {
	m.unregister <- client
}

func (m *Manager) Broadcast(message []byte) {
	m.broadcast <- message
}

func (m *Manager) SendToUser(userID string, message []byte) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if clientIDs, ok := m.userClients[userID]; ok {
		for _, clientID := range clientIDs {
			if client, exists := m.clients[clientID]; exists {
				select {
				case client.Send <- message:
				default:
					go m.Unregister(client)
				}
			}
		}
	}
}

func (m *Manager) GetClientCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clients)
}

func (m *Manager) GetUserClientCount(userID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if clients, ok := m.userClients[userID]; ok {
		return len(clients)
	}
	return 0
}

func (m *Manager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"total_connections":   m.stats.totalConnections,
		"current_connections": m.stats.currentConnections,
		"max_connections":     m.stats.maxConnections,
		"unique_users":        len(m.userClients),
	}
}

func (m *Manager) monitorStats() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			stats := m.GetStats()
			log.Printf("WebSocket Stats - Current: %d, Max: %d, Users: %d",
				stats["current_connections"], stats["max_connections"], stats["unique_users"])
		}
	}
}
