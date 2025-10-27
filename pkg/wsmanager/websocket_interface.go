package wsmanager

import (
	"io"
	"net"
	"time"
)

// WebSocketConn 定义 WebSocket 连接的标准接口
type WebSocketConn interface {
	// 基本消息操作
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
	Close() error

	// 超时控制
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error

	// 处理器设置
	SetPongHandler(h func(appData string) error)
	SetPingHandler(h func(appData string) error)
	SetCloseHandler(h func(code int, text string) error)

	// 其他必要方法
	SetReadLimit(limit int64)
	WriteControl(messageType int, data []byte, deadline time.Time) error
	NextReader() (messageType int, r io.Reader, err error)
	NextWriter(messageType int) (io.WriteCloser, error)

	// 连接信息
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	Subprotocol() string
}
