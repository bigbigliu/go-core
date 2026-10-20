package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestIPFilterMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	limiter := NewMemoryLimiter(2, time.Minute)
	defer limiter.Close()

	router := gin.New()
	router.Use(limiter.Middleware())
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	doRequest := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.0.2.10:1234"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	if w := doRequest(); w.Code != http.StatusOK {
		t.Fatalf("request 1 status = %d", w.Code)
	}
	if w := doRequest(); w.Code != http.StatusOK {
		t.Fatalf("request 2 status = %d", w.Code)
	}
	w := doRequest()
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("request 3 status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}
	if got := w.Header().Get("X-RateLimit-Remaining"); got != "0" {
		t.Errorf("X-RateLimit-Remaining = %q, want 0", got)
	}
}

func TestMemoryLimiterCloseIdempotent(t *testing.T) {
	limiter := NewMemoryLimiter(1, time.Minute)
	limiter.Close()
	limiter.Close()
}
