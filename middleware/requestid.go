package middleware

import (
	"github.com/bigbigliu/go-core/idgen"
	"github.com/bigbigliu/go-core/logger"
	"github.com/gin-gonic/gin"
)

// RequestIDMiddleware 为每个请求生成 Request ID，并同时写入响应头、
// gin.Context 和 request.Context，便于日志链路追踪。
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(logger.RequestIDKey)
		if requestID == "" {
			requestID = idgen.GenUniversalID()
		}

		c.Header(logger.RequestIDKey, requestID)
		c.Set(logger.RequestIDKey, requestID)
		c.Request = c.Request.WithContext(logger.ContextWithRequestID(c.Request.Context(), requestID))

		c.Next()
	}
}
