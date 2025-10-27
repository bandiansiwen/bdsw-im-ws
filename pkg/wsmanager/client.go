package wsmanager

import (
	"time"
)

// Client WebSocket客户端
type Client struct {
	ID          string
	Conn        WebSocketConn
	Send        chan []byte
	UserID      string
	Platform    string
	IP          string
	ConnectedAt time.Time
	Metadata    map[string]interface{}
}

func NewClient(id string, conn WebSocketConn, userID string) *Client {
	return &Client{
		ID:          id,
		Conn:        conn,
		Send:        make(chan []byte, 256),
		UserID:      userID,
		ConnectedAt: time.Now(),
		Metadata:    make(map[string]interface{}),
	}
}

// SetMetadata 设置元数据
func (c *Client) SetMetadata(key string, value interface{}) {
	c.Metadata[key] = value
}

// GetMetadata 获取元数据
func (c *Client) GetMetadata(key string) (interface{}, bool) {
	value, exists := c.Metadata[key]
	return value, exists
}

// GetUserInfo 获取用户信息
func (c *Client) GetUserInfo() (*UserInfo, bool) {
	if userInfo, exists := c.Metadata["user_info"]; exists {
		if info, ok := userInfo.(*UserInfo); ok {
			return info, true
		}
	}
	return nil, false
}
