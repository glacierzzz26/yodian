// Package middleware 鉴权中间件（设计 16.2）。
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/yodian/server/internal/auth"
	"github.com/yodian/server/internal/config"
	"github.com/yodian/server/internal/pkg/respond"
)

// CustomerAuth 顾客端 JWT 校验，成功后注入 cid/sid/ch 上下文
func CustomerAuth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := auth.ParseCustomer(cfg.JWTCustomerSecret, bearer(c))
		if err != nil {
			respond.Err(c, respond.ErrUnauthorized)
			c.Abort()
			return
		}
		c.Set("cid", claims.CID)
		c.Set("sid", claims.SID)
		c.Set("ch", claims.CH)
		c.Next()
	}
}

// RequireRole 角色白名单：不在列返回 403（16.2 权限矩阵的运行时执行；矩阵表在 2.4 定稿）
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("role")
		for _, r := range roles {
			if role == r {
				c.Next()
				return
			}
		}
		respond.Err(c, respond.ErrForbidden)
		c.Abort()
	}
}

// StaffAuth 商家端 JWT 校验（阶段 2.3 起挂载清台/核销/退款；角色矩阵在 2.4），注入 oid/role/sid
func StaffAuth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := auth.ParseStaff(cfg.JWTStaffSecret, bearer(c))
		if err != nil {
			respond.Err(c, respond.ErrUnauthorized)
			c.Abort()
			return
		}
		c.Set("oid", claims.OID)
		c.Set("role", claims.Role)
		c.Set("sid", claims.SID)
		c.Next()
	}
}

func bearer(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(h[7:])
}
