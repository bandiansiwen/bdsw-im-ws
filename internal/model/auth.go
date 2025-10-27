package model

type UserInfo struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email,omitempty"`
	Avatar   string `json:"avatar,omitempty"`
}

type VerifyTokenResponse struct {
	Valid   bool     `json:"valid"`
	User    UserInfo `json:"user,omitempty"`
	Message string   `json:"message,omitempty"`
}
