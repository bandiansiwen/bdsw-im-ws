package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 简单的认证示例
		// 在实际项目中，你应该使用JWT或其他认证方式

		token := c.GetHeader("Authorization")
		if token == "" {
			token = c.Query("token")
		}

		if token != "" && strings.HasPrefix(token, "Bearer ") {
			token = strings.TrimPrefix(token, "Bearer ")
		}

		// 示例：从token中解析用户ID
		// 在实际项目中，你应该验证token的签名和有效期
		var userID string
		if token != "" {
			// 这里应该是实际的token解析逻辑
			userID = "user_" + token // 示例
		} else {
			// 如果没有token，生成匿名用户ID
			userID = "anonymous_" + c.ClientIP()
		}

		c.Set("userID", userID)
		c.Next()
	}
}
