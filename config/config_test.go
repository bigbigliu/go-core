package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "settings.yml")
	content := `
app:
  host: 127.0.0.1
  port: 8080
  mode: release
db:
  host: 127.0.0.1
  port: 3307
  user: root
  password: secret
  name: demo
redis:
  addr: 127.0.0.1
  port: 6380
  pwd: ""
  db: 1
logger:
  path: ./logs
  level: debug
jwt:
  secret: test-secret
  timeout: 3600
`
	if err := os.WriteFile(filename, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := Load(filename); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	cfg := GetConfig()
	if cfg == nil {
		t.Fatal("GetConfig() returned nil")
	}
	if cfg.App.Port != 8080 || cfg.App.Mode != "release" {
		t.Errorf("unexpected app config: %+v", cfg.App)
	}
	if cfg.Db.Password != "secret" || cfg.Db.MaxIdleConnections != 10 || cfg.Db.MaxOpenConnections != 100 {
		t.Errorf("unexpected db config: %+v", cfg.Db)
	}
	if cfg.Redis.Port != 6380 || cfg.Redis.Db != 1 {
		t.Errorf("unexpected redis config: %+v", cfg.Redis)
	}
	if cfg.Logger.Path != "./logs" || cfg.Logger.Level != "debug" {
		t.Errorf("unexpected logger config: %+v", cfg.Logger)
	}
	if cfg.Jwt.Timeout != 3600 {
		t.Errorf("unexpected jwt config: %+v", cfg.Jwt)
	}
}

func TestLoadAppliesDefaults(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "settings.yml")
	if err := os.WriteFile(filename, []byte("jwt:\n  secret: s\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := Load(filename); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	cfg := GetConfig()
	if cfg.Db.Port != 3306 || cfg.Db.MaxIdleConnections != 10 || cfg.Db.MaxOpenConnections != 100 {
		t.Errorf("db defaults not applied: %+v", cfg.Db)
	}
	if cfg.Redis.Port != 6379 {
		t.Errorf("redis default port not applied: %+v", cfg.Redis)
	}
	if cfg.Logger.Path != "./log" || cfg.Logger.Level != "info" {
		t.Errorf("logger defaults not applied: %+v", cfg.Logger)
	}
	if cfg.Jwt.Timeout != 7200 {
		t.Errorf("jwt default timeout not applied: %+v", cfg.Jwt)
	}
	if cfg.App == nil || cfg.Db == nil || cfg.Redis == nil {
		t.Error("nil config section after defaults")
	}
	if cfg.Mongo == nil || cfg.Mongo.Host != "127.0.0.1" || cfg.Mongo.Port != 27017 {
		t.Errorf("mongo defaults not applied: %+v", cfg.Mongo)
	}
	if cfg.Mongo.ConnectTimeout != 10 || cfg.Mongo.ServerSelectionTimeout != 10 {
		t.Errorf("mongo timeout defaults not applied: %+v", cfg.Mongo)
	}
}

func TestLoadMultipleDatabases(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "settings.yml")
	content := `
db:
  host: 127.0.0.1
  name: primary
databases:
  orders:
    host: 127.0.0.1
    name: orders
  report:
    host: 127.0.0.2
    port: 3307
    user: reporter
    password: secret
    name: report
    max_idle_connections: 5
    max_open_connections: 50
  analytics:
    host: 127.0.0.3
    name: analytics
mongo:
  host: mongo.local
  port: 27018
  database: core
  username: alice
  password: secret
  connect_timeout: 3
`
	if err := os.WriteFile(filename, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Load(filename); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	cfg := GetConfig()
	if cfg.Db.Port != 3306 || cfg.Db.MaxIdleConnections != 10 || cfg.Db.MaxOpenConnections != 100 {
		t.Errorf("primary db defaults not applied: %+v", cfg.Db)
	}
	if len(cfg.Databases) != 3 {
		t.Fatalf("Databases length = %d, want 3", len(cfg.Databases))
	}
	orders := cfg.Databases["orders"]
	if orders == nil || orders.Name != "orders" || orders.Port != 3306 {
		t.Errorf("unexpected orders config: %+v", orders)
	}
	report := cfg.Databases["report"]
	if report == nil || report.Port != 3307 || report.MaxIdleConnections != 5 || report.MaxOpenConnections != 50 {
		t.Errorf("unexpected report config: %+v", report)
	}
	analytics := cfg.Databases["analytics"]
	if analytics == nil || analytics.Name != "analytics" || analytics.Port != 3306 {
		t.Errorf("unexpected analytics config: %+v", analytics)
	}
	if cfg.Mongo.Host != "mongo.local" || cfg.Mongo.Port != 27018 || cfg.Mongo.ConnectTimeout != 3 || cfg.Mongo.Username != "alice" {
		t.Errorf("unexpected mongo config: %+v", cfg.Mongo)
	}
}

func TestLoadMissingFile(t *testing.T) {
	if err := Load(filepath.Join(t.TempDir(), "missing.yml")); err == nil {
		t.Fatal("Load() expected error for missing file")
	}
}

func TestSettingsDevSample(t *testing.T) {
	if err := Load("settings-dev.yml"); err != nil {
		t.Fatalf("load settings-dev.yml: %v", err)
	}
	cfg := GetConfig()
	if len(cfg.Databases) < 3 {
		t.Fatalf("Databases length = %d, want at least 3", len(cfg.Databases))
	}
	for _, name := range []string{"orders", "report", "analytics"} {
		if cfg.Databases[name] == nil {
			t.Errorf("missing database config: %s", name)
		}
	}
	if cfg.Mongo == nil || cfg.Mongo.Database != "core" {
		t.Errorf("unexpected mongo config: %+v", cfg.Mongo)
	}
}

func TestSetConfigAppliesDefaults(t *testing.T) {
	cfg := &Config{}
	SetConfig(cfg)

	if cfg.App == nil || cfg.Db == nil || cfg.Redis == nil || cfg.Logger == nil || cfg.Mongo == nil {
		t.Fatal("SetConfig should fill nil sections")
	}
	if cfg.Db.MaxIdleConnections != 10 || cfg.Db.MaxOpenConnections != 100 {
		t.Errorf("db defaults not applied: %+v", cfg.Db)
	}
	if cfg.Logger.Path != "./log" || cfg.Logger.Level != "info" {
		t.Errorf("logger defaults not applied: %+v", cfg.Logger)
	}
	if cfg.Mongo.Port != 27017 || cfg.Mongo.Host != "127.0.0.1" {
		t.Errorf("mongo defaults not applied: %+v", cfg.Mongo)
	}
}
