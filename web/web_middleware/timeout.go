package web_middleware

import (
	"net/http"
	"time"

	"github.com/gin-contrib/timeout"
	"github.com/gin-gonic/gin"
)

// TimeoutMiddleware 接口超时中间件
func TimeoutMiddleware(reqTimeout time.Duration) gin.HandlerFunc {
	return timeout.New(
		timeout.WithTimeout(reqTimeout),
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
