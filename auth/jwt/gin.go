package jwt

import (
	"net/http"
	"strings"

	"github.com/bigbigliu/go-core/database/redis"
	"github.com/bigbigliu/go-core/response"
	"github.com/gin-gonic/gin"
)

const accessTokenKeyPrefix = "coreos:"

// TokenVerify 校验 JWT 是否合法且仍存在于 Redis 中。
func (h *CoreJWT) TokenVerify() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearerToken(c.GetHeader("Authorization"))
		if token == "" {
			unauthorized(c, "token不能为空")
			return
		}

		claims, err := h.ParseToken(token)
		if err != nil || claims == nil || claims.Username == "" {
			unauthorized(c, "token无效或已过期")
			return
		}

		if redis.Redisclient == nil {
			c.JSON(http.StatusInternalServerError, response.FailMsg("redis未初始化"))
			c.Abort()
			return
		}

		tokenKey := accessTokenKeyPrefix + claims.Username + ":accesstoken"
		storedToken, err := redis.Redisclient.Get(c.Request.Context(), tokenKey).Result()
		if err != nil || storedToken != token {
			unauthorized(c, "token已失效")
			return
		}

		c.Set("username", claims.Username)
		c.Next()
	}
}

func extractBearerToken(header string) string {
	parts := strings.Fields(header)
	if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
		return parts[1]
	}
	return strings.TrimSpace(header)
}

func unauthorized(c *gin.Context, msg string) {
	c.JSON(http.StatusUnauthorized, response.FailMsg(msg))
	c.Abort()
}
