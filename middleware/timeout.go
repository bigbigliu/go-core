package middleware

import (
	"net/http"
	"time"

	"github.com/bigbigliu/go-core/response"
	"github.com/gin-contrib/timeout"
	"github.com/gin-gonic/gin"
)

// TimeoutMiddleware 接口超时中间件。
// 注意：超时后业务 handler 仍可能继续执行，handler 内部应使用带超时的 context。
func TimeoutMiddleware(reqTimeout time.Duration) gin.HandlerFunc {
	return timeout.New(
		timeout.WithTimeout(reqTimeout),
		timeout.WithResponse(TimeoutResponse),
	)
}

// TimeoutResponse 超时响应
func TimeoutResponse(c *gin.Context) {
	c.JSON(http.StatusRequestTimeout, response.FailMsg("timeout"))
}
