package wsmanager

import (
	"time"
)

type Client struct {
	ID          string
	Conn        WebSocketConn // 使用接口
	Send        chan []byte
	UserID      string
	Platform    string
	IP          string
	ConnectedAt time.Time
}

func NewClient(id string, conn WebSocketConn, userID string) *Client {
	return &Client{
		ID:          id,
		Conn:        conn,
		Send:        make(chan []byte, 256),
		UserID:      userID,
		ConnectedAt: time.Now(),
	}
}
