package middleware

import (
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// CorsConfig 跨域配置
type CorsConfig struct {
	// AllowOrigins 允许的 Origin 列表。使用 "*" 表示允许任意 Origin，
	// 此时 AllowCredentials 必须为 false。
	AllowOrigins []string
	AllowMethods []string
	AllowHeaders []string
	// AllowCredentials 是否允许浏览器携带 Cookie/Authorization。
	AllowCredentials bool
	MaxAge           time.Duration
}

// DefaultCorsConfig 返回默认跨域配置。
// 默认允许任意 Origin 的公开跨域请求，但不允许携带凭证。
func DefaultCorsConfig() CorsConfig {
	return CorsConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization", "Origin", "X-Auth-Token", "X-Request-ID", "X-Requested-With"},
		AllowCredentials: false,
		MaxAge:           24 * time.Hour,
	}
}

// CorsMiddleware 使用默认配置的跨域中间件
func CorsMiddleware() gin.HandlerFunc {
	return CorsMiddlewareWithConfig(DefaultCorsConfig())
}

// CorsMiddlewareWithConfig 使用指定配置的跨域中间件。
// 当 AllowCredentials 为 true 时，必须配置明确的 AllowOrigins，"*" 将被忽略。
func CorsMiddlewareWithConfig(cfg CorsConfig) gin.HandlerFunc {
	if len(cfg.AllowMethods) == 0 {
		cfg.AllowMethods = DefaultCorsConfig().AllowMethods
	}
	if len(cfg.AllowHeaders) == 0 {
		cfg.AllowHeaders = DefaultCorsConfig().AllowHeaders
	}
	if cfg.MaxAge <= 0 {
		cfg.MaxAge = 24 * time.Hour
	}
	allowAll := slices.Contains(cfg.AllowOrigins, "*") && !cfg.AllowCredentials

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowedOrigin := ""
		switch {
		case origin == "":
			// 非浏览器跨域请求，无需 CORS 响应头
		case allowAll:
			allowedOrigin = "*"
		case slices.Contains(cfg.AllowOrigins, origin):
			allowedOrigin = origin
		}

		if allowedOrigin != "" {
			c.Header("Access-Control-Allow-Origin", allowedOrigin)
			c.Header("Vary", "Origin")
			if cfg.AllowCredentials {
				c.Header("Access-Control-Allow-Credentials", "true")
			}
		}

		c.Header("Access-Control-Allow-Methods", strings.Join(cfg.AllowMethods, ", "))
		c.Header("Access-Control-Allow-Headers", strings.Join(cfg.AllowHeaders, ", "))
		c.Header("Access-Control-Max-Age", strconv.Itoa(int(cfg.MaxAge.Seconds())))

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
