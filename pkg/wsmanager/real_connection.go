package wsmanager

import (
	"github.com/gorilla/websocket"
)

// RealWebSocketConn 包装真实的 *websocket.Conn 以实现我们的接口
type RealWebSocketConn struct {
	*websocket.Conn
}

// NewRealWebSocketConn 创建真实连接的适配器
func NewRealWebSocketConn(conn *websocket.Conn) WebSocketConn {
	return &RealWebSocketConn{Conn: conn}
}
