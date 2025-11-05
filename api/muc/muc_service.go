package muc

import (
	"bdsw-im-ws/api/common"
	"bdsw-im-ws/api/ima_gateway"
	"context"
)

// ================================
// MUC 认证服务接口
// ================================

type MUCService interface {
	// Token 验证
	ValidateToken(ctx context.Context, req *TokenValidateRequest) (*TokenValidateResponse, error)
	// 踢用户下线
	KickUser(ctx context.Context, req *ima_gateway.KickUserRequest) (*common.BaseResponse, error)
	// 刷新 Token
	RefreshToken(ctx context.Context, req *RefreshTokenRequest) (*RefreshTokenResponse, error)
	// 获取用户权限
	GetUserPermissions(ctx context.Context, req *GetUserPermissionsRequest) (*GetUserPermissionsResponse, error)
	// 用户登录
	Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error)
	// 用户登出
	Logout(ctx context.Context, req *LogoutRequest) (*common.BaseResponse, error)
}
