package bootstrap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func writeTestConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.yml")
	content := `
app:
  host: 127.0.0.1
  port: 0
  mode: test
logger:
  path: ` + filepath.Join(dir, "log") + `
  level: info
jwt:
  secret: test-secret
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestNewWithConfigPath(t *testing.T) {
	app, err := New(context.Background(), Options{
		ConfigPath:    writeTestConfig(t),
		DisableLogger: true,
		Host:          "127.0.0.1",
		Port:          9090,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = app.Close() }()

	if app.Config == nil || app.Router == nil {
		t.Fatal("config or router is nil")
	}
	if app.Config.App.Port != 9090 || app.Config.App.Host != "127.0.0.1" {
		t.Errorf("address overrides not applied: %+v", app.Config.App)
	}
	if app.Gin() != app.Router {
		t.Error("Gin() should return the initialized router")
	}

	app.Router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	app.Router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("missing X-Request-ID header")
	}
}

func TestNewComponentValidation(t *testing.T) {
	path := writeTestConfig(t)

	if _, err := New(context.Background(), Options{ConfigPath: path, DisableLogger: true, InitMySQL: true}); err == nil {
		t.Error("New(InitMySQL) expected error without db config")
	}
	if _, err := New(context.Background(), Options{ConfigPath: path, DisableLogger: true, InitRedis: true}); err == nil {
		t.Error("New(InitRedis) expected error without redis config")
	}
}

func TestNewMongoValidation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.yml")
	content := `
app:
  host: 127.0.0.1
  port: 0
  mode: test
logger:
  path: ` + filepath.Join(dir, "log") + `
  level: info
jwt:
  secret: test-secret
mongo:
  uri: "://invalid"
  database: core
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := New(context.Background(), Options{ConfigPath: path, DisableLogger: true, InitMongo: true}); err == nil {
		t.Error("New(InitMongo) expected error for invalid URI")
	}
}
