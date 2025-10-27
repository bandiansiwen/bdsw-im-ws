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

// 注意：由于嵌入了 *websocket.Conn，所有 WebSocketConn 接口的方法都会自动实现
// 不需要额外编写任何代码！
