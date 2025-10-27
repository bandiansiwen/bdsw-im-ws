package wsmanager

import (
	"bytes"
	"io"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// MockWebSocketConn 模拟 WebSocket 连接，用于测试
type MockWebSocketConn struct {
	writeMessages [][]byte
	writeTypes    []int
	readMessages  [][]byte
	readTypes     []int
	readIndex     int
	closed        bool

	// 配置
	readDeadline  time.Time
	writeDeadline time.Time
	readLimit     int64

	// 处理器
	pongHandler  func(string) error
	pingHandler  func(string) error
	closeHandler func(int, string) error

	// 同步
	mu       sync.RWMutex
	readChan chan struct{}
}

// NewMockWebSocketConn 创建模拟连接（返回接口类型）
func NewMockWebSocketConn() WebSocketConn {
	return &MockWebSocketConn{
		writeMessages: make([][]byte, 0),
		writeTypes:    make([]int, 0),
		readMessages:  make([][]byte, 0),
		readTypes:     make([]int, 0),
		readChan:      make(chan struct{}, 100),
	}
}

// ReadMessage 实现读取消息
func (m *MockWebSocketConn) ReadMessage() (int, []byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return 0, nil, io.EOF
	}

	if m.readIndex >= len(m.readMessages) {
		return 0, nil, &websocket.CloseError{Code: websocket.CloseNormalClosure}
	}

	messageType := m.readTypes[m.readIndex]
	message := m.readMessages[m.readIndex]
	m.readIndex++

	return messageType, message, nil
}

// WriteMessage 实现写入消息
func (m *MockWebSocketConn) WriteMessage(messageType int, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return io.EOF
	}

	m.writeMessages = append(m.writeMessages, data)
	m.writeTypes = append(m.writeTypes, messageType)
	return nil
}

// Close 关闭连接
func (m *MockWebSocketConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.closed = true
	close(m.readChan)
	return nil
}

// 其他接口方法的简单实现
func (m *MockWebSocketConn) SetReadDeadline(t time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readDeadline = t
	return nil
}

func (m *MockWebSocketConn) SetWriteDeadline(t time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writeDeadline = t
	return nil
}

func (m *MockWebSocketConn) SetPongHandler(h func(string) error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pongHandler = h
}

func (m *MockWebSocketConn) SetPingHandler(h func(string) error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pingHandler = h
}

func (m *MockWebSocketConn) SetCloseHandler(h func(int, string) error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closeHandler = h
}

func (m *MockWebSocketConn) SetReadLimit(limit int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readLimit = limit
}

func (m *MockWebSocketConn) WriteControl(messageType int, data []byte, deadline time.Time) error {
	// 简化实现，直接调用 WriteMessage
	return m.WriteMessage(messageType, data)
}

func (m *MockWebSocketConn) NextReader() (int, io.Reader, error) {
	messageType, data, err := m.ReadMessage()
	if err != nil {
		return 0, nil, err
	}
	return messageType, bytes.NewReader(data), nil
}

func (m *MockWebSocketConn) NextWriter(messageType int) (io.WriteCloser, error) {
	return &mockWriter{conn: m, messageType: messageType}, nil
}

func (m *MockWebSocketConn) LocalAddr() net.Addr {
	return &mockAddr{network: "tcp", addr: "localhost:8080"}
}

func (m *MockWebSocketConn) RemoteAddr() net.Addr {
	return &mockAddr{network: "tcp", addr: "client:12345"}
}

func (m *MockWebSocketConn) Subprotocol() string {
	return ""
}

// 测试辅助方法
func (m *MockWebSocketConn) AddReadMessage(messageType int, data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.readMessages = append(m.readMessages, data)
	m.readTypes = append(m.readTypes, messageType)

	select {
	case m.readChan <- struct{}{}:
	default:
		// 通道满，忽略
	}
}

func (m *MockWebSocketConn) GetWriteMessages() [][]byte {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([][]byte, len(m.writeMessages))
	copy(result, m.writeMessages)
	return result
}

func (m *MockWebSocketConn) GetWriteMessageTypes() []int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]int, len(m.writeTypes))
	copy(result, m.writeTypes)
	return result
}

func (m *MockWebSocketConn) IsClosed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.closed
}

// 辅助类型
type mockWriter struct {
	conn        *MockWebSocketConn
	messageType int
	buffer      bytes.Buffer
}

func (w *mockWriter) Write(p []byte) (int, error) {
	return w.buffer.Write(p)
}

func (w *mockWriter) Close() error {
	return w.conn.WriteMessage(w.messageType, w.buffer.Bytes())
}

type mockAddr struct {
	network string
	addr    string
}

func (m *mockAddr) Network() string { return m.network }
func (m *mockAddr) String() string  { return m.addr }
