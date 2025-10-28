package model

type UserInfo struct {
	UserID   string `json:"userId"`
	Username string `json:"userName"`
}

type VerifyTokenResponse struct {
	Code    int32    `json:"code"`
	User    UserInfo `json:"user,omitempty"`
	Message string   `json:"message,omitempty"`
}
