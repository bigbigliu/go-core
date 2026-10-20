package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCorsMiddlewareDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CorsMiddleware())
	router.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Allow-Origin = %q, want *", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Errorf("Allow-Credentials = %q, want empty", got)
	}
	if got := w.Header().Get("Vary"); got != "Origin" {
		t.Errorf("Vary = %q", got)
	}
}

func TestCorsMiddlewareWithCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := DefaultCorsConfig()
	cfg.AllowOrigins = []string{"https://example.com"}
	cfg.AllowCredentials = true

	router := gin.New()
	router.Use(CorsMiddlewareWithConfig(cfg))
	router.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Errorf("Allow-Origin = %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("Allow-Credentials = %q", got)
	}
}

func TestCorsMiddlewareRejectsUnknownOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := DefaultCorsConfig()
	cfg.AllowOrigins = []string{"https://allowed.example"}

	router := gin.New()
	router.Use(CorsMiddlewareWithConfig(cfg))
	router.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://evil.example")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Allow-Origin = %q, want empty", got)
	}
}
