package web_middleware
package web_middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"context"
)

func setupRedis() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})
}

func TestIPFilterWithRedisMiddleware(t *testing.T) {
	redisClient := setupRedis()
	defer redisClient.FlushDB(context.Background())

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(IPFilterWithRedisMiddleware(redisClient, 3, time.Second*2))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"msg": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "1.2.3.4:12345"

	// 模拟同一IP多次请求
	var resp *httptest.ResponseRecorder
	for i := 1; i <= 5; i++ {
		resp = httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if i <= 3 {
			assert.Equal(t, 200, resp.Code)
			assert.Equal(t, "ok", resp.Body.String()[8:10])
		} else {
			assert.Equal(t, http.StatusTooManyRequests, resp.Code)
			assert.Contains(t, resp.Body.String(), "Too many requests")
		}
	}

	// 等待窗口过期后再次请求
	time.Sleep(3 * time.Second)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	assert.Equal(t, 200, resp.Code)
}
