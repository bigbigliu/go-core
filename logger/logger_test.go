package logger

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitializeLogger(t *testing.T) {
	dir := t.TempDir()
	err := InitializeLogger(&CoreLog{
		LogDir:        dir,
		LogLevel:      "debug",
		ConsoleOutPut: false,
		ServiceName:   "test-service",
	})
	if err != nil {
		t.Fatalf("InitializeLogger() error = %v", err)
	}
	if Logger == nil {
		t.Fatal("Logger is nil")
	}

	ctx := ContextWithRequestID(context.Background(), "req-123")
	Logger.WithOptions(WithContext(ctx)).Info("hello")
	if err := Sync(); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "app.log"))
	if err != nil {
		t.Fatal(err)
	}
	logContent := string(data)
	if !strings.Contains(logContent, "req-123") {
		t.Errorf("log does not contain request id: %s", logContent)
	}
	if !strings.Contains(logContent, "test-service") {
		t.Errorf("log does not contain service name: %s", logContent)
	}
}

func TestWithContextWithoutRequestID(t *testing.T) {
	option := WithContext(context.Background())
	if option == nil {
		t.Fatal("WithContext() returned nil option")
	}
}
