package wsmanager

// UserInfo 用户信息
type UserInfo struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Email    string   `json:"email,omitempty"`
	Avatar   string   `json:"avatar,omitempty"`
	Roles    []string `json:"roles,omitempty"`
}
