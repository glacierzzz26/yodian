// 限流（设计 16.3）：Redis 固定窗口，命中返回 429 信封。
// 维度：IP（默认）/ customer_id / operator_id，按 keyFunc 注入。
package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/yodian/server/internal/pkg/respond"
)

// RateLimit 按 IP 在 window 内最多 limit 次（默认维度）
func RateLimit(rdb *redis.Client, prefix string, window time.Duration, limit int) gin.HandlerFunc {
	return RateLimitBy(rdb, prefix, window, limit, func(c *gin.Context) string {
		return c.ClientIP()
	})
}

// RateLimitBy 按 keyFunc 计算限流键（如 customer_id / operator_id，16.3 表）
func RateLimitBy(rdb *redis.Client, prefix string, window time.Duration, limit int, keyFunc func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := fmt.Sprintf("rl:%s:%s:%d", prefix, keyFunc(c), window/time.Second)
		ctx := c.Request.Context()
		n, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			c.Next() // Redis 异常放行（降级为不限制），避免业务被限流拖垮
			return
		}
		if n == 1 {
			rdb.Expire(ctx, key, window)
		}
		if n > int64(limit) {
			respond.Err(c, respond.ErrTooManyReqs)
			c.Abort()
			return
		}
		c.Next()
	}
}
