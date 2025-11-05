package service

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type UserConnection struct {
	UserID    string
	DeviceID  string
	Conn      *websocket.Conn
	LoginTime int64
}

type UserConnectionManager struct {
	mu sync.RWMutex
	// userID -> deviceID -> connection
	users map[string]map[string]*UserConnection
}

func NewUserConnectionManager() *UserConnectionManager {
	return &UserConnectionManager{
		users: make(map[string]map[string]*UserConnection),
	}
}

// Register 注册用户连接 - 返回指针，避免复制
func (m *UserConnectionManager) Register(userID, deviceID string, conn *websocket.Conn) *UserConnection {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.users[userID]; !exists {
		m.users[userID] = make(map[string]*UserConnection)
	}

	userConn := &UserConnection{
		UserID:    userID,
		DeviceID:  deviceID,
		Conn:      conn, // 直接存储指针
		LoginTime: time.Now().Unix(),
	}

	m.users[userID][deviceID] = userConn
	return userConn
}

// Unregister 注销用户连接
func (m *UserConnectionManager) Unregister(userID, deviceID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if devices, exists := m.users[userID]; exists {
		delete(devices, deviceID)
		if len(devices) == 0 {
			delete(m.users, userID)
		}
	}
}

// Get 获取连接 - 返回指针
func (m *UserConnectionManager) Get(userID, deviceID string) *websocket.Conn {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if devices, exists := m.users[userID]; exists {
		if userConn, exists := devices[deviceID]; exists {
			return userConn.Conn // 返回指针
		}
	}
	return nil
}

// GetConnection 获取完整的连接信息
func (m *UserConnectionManager) GetConnection(userID, deviceID string) *UserConnection {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if devices, exists := m.users[userID]; exists {
		return devices[deviceID]
	}
	return nil
}

// GetUserDevices 获取用户的所有设备ID
func (m *UserConnectionManager) GetUserDevices(userID string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if devices, exists := m.users[userID]; exists {
		deviceIDs := make([]string, 0, len(devices))
		for deviceID := range devices {
			deviceIDs = append(deviceIDs, deviceID)
		}
		return deviceIDs
	}
	return nil
}

// GetAllUserDevices 获取所有用户的设备信息
func (m *UserConnectionManager) GetAllUserDevices() map[string][]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string][]string)
	for userID, devices := range m.users {
		deviceIDs := make([]string, 0, len(devices))
		for deviceID := range devices {
			deviceIDs = append(deviceIDs, deviceID)
		}
		result[userID] = deviceIDs
	}
	return result
}

// GetOnlineCount 获取在线用户数
func (m *UserConnectionManager) GetOnlineCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, devices := range m.users {
		count += len(devices)
	}
	return count
}

// BroadcastToUser 向用户的所有设备广播 - 使用指针
func (m *UserConnectionManager) BroadcastToUser(userID string, handler func(deviceID string, conn *websocket.Conn)) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if devices, exists := m.users[userID]; exists {
		for deviceID, userConn := range devices {
			handler(deviceID, userConn.Conn) // 传递指针
		}
	}
}

// Broadcast 向所有用户广播 - 使用指针
func (m *UserConnectionManager) Broadcast(handler func(userID, deviceID string, conn *websocket.Conn)) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for userID, devices := range m.users {
		for deviceID, userConn := range devices {
			handler(userID, deviceID, userConn.Conn) // 传递指针
		}
	}
}

// CloseAll 关闭所有连接
func (m *UserConnectionManager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for userID, devices := range m.users {
		for deviceID, userConn := range devices {
			userConn.Conn.Close()
			delete(devices, deviceID)
		}
		delete(m.users, userID)
	}
}
