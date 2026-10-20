package mongodb

import (
	"context"
	"testing"
	"time"
)

func TestConfigDSN(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{"defaults", Config{}, "mongodb://127.0.0.1:27017"},
		{"host and port", Config{Host: "db.local", Port: 27018}, "mongodb://db.local:27018"},
		{"ipv6", Config{Host: "::1", Port: 27018}, "mongodb://[::1]:27018"},
		{"uri wins", Config{URI: "mongodb://example:27017", Host: "ignored"}, "mongodb://example:27017"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.dsn(); got != tt.want {
				t.Errorf("dsn() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConfigClientOptions(t *testing.T) {
	cfg := Config{
		URI:                    "mongodb://127.0.0.1:27017",
		Username:               "alice",
		Password:               "secret",
		AuthSource:             "admin",
		AppName:                "demo",
		MaxPoolSize:            20,
		MinPoolSize:            2,
		ConnectTimeout:         time.Second,
		ServerSelectionTimeout: 2 * time.Second,
	}

	opts := cfg.clientOptions()
	if opts.Auth == nil || opts.Auth.Username != "alice" || opts.Auth.AuthSource != "admin" || !opts.Auth.PasswordSet {
		t.Errorf("unexpected auth options: %+v", opts.Auth)
	}
	if opts.AppName == nil || *opts.AppName != "demo" {
		t.Errorf("unexpected app name: %v", opts.AppName)
	}
	if opts.MaxPoolSize == nil || *opts.MaxPoolSize != 20 {
		t.Errorf("unexpected max pool size: %v", opts.MaxPoolSize)
	}
	if opts.MinPoolSize == nil || *opts.MinPoolSize != 2 {
		t.Errorf("unexpected min pool size: %v", opts.MinPoolSize)
	}
}

func TestNewClientInvalidURI(t *testing.T) {
	_, err := NewClient(context.Background(), Config{
		URI:            "://invalid",
		ConnectTimeout: 100 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("NewClient() expected error for invalid URI")
	}
}

func TestNilClient(t *testing.T) {
	var client *Client
	if client.Raw() != nil {
		t.Error("Raw() should be nil")
	}
	if client.DefaultDatabase() != "" {
		t.Error("DefaultDatabase() should be empty")
	}
	if _, err := client.Database(); err == nil {
		t.Error("Database() expected error")
	}
	if err := client.Ping(context.Background()); err == nil {
		t.Error("Ping() expected error")
	}
	if err := client.Close(context.Background()); err != nil {
		t.Errorf("Close() on nil client error = %v", err)
	}
}
