package web_middleware

import (
	"github.com/gin-contrib/timeout"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

// TimeoutMiddleware 接口超时中间件
func TimeoutMiddleware(reqTimeout time.Duration) gin.HandlerFunc {
	return timeout.New(
		timeout.WithTimeout(reqTimeout),
		timeout.WithHandler(func(c *gin.Context) {
			c.Next()
		}),
		timeout.WithResponse(TimeoutResponse),
	)
}

// TimeoutResponse 超时响应
func TimeoutResponse(c *gin.Context) {
	c.JSON(http.StatusRequestTimeout, gin.H{
		"code": http.StatusRequestTimeout,
		"msg":  "timeout",
	})
}
