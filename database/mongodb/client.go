// Package mongodb provides MongoDB connection helpers built on the official
// go.mongodb.org/mongo-driver/v2 driver.
package mongodb

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

const (
	defaultHost = "127.0.0.1"
	defaultPort = 27017
	// DefaultConnectTimeout 默认连接/健康检查超时
	DefaultConnectTimeout = 10 * time.Second
)

// Config MongoDB 连接配置。
// URI 非空时优先使用 URI，Host/Port 仅在 URI 为空时生效。
type Config struct {
	URI                    string        `json:"uri" yaml:"uri"`
	Host                   string        `json:"host" yaml:"host"`
	Port                   int           `json:"port" yaml:"port"`
	Database               string        `json:"database" yaml:"database"`
	Username               string        `json:"username" yaml:"username"`
	Password               string        `json:"password" yaml:"password"`
	AuthSource             string        `json:"auth_source" yaml:"auth_source"`
	AppName                string        `json:"app_name" yaml:"app_name"`
	MaxPoolSize            uint64        `json:"max_pool_size" yaml:"max_pool_size"`
	MinPoolSize            uint64        `json:"min_pool_size" yaml:"min_pool_size"`
	ConnectTimeout         time.Duration `json:"connect_timeout" yaml:"connect_timeout"`
	ServerSelectionTimeout time.Duration `json:"server_selection_timeout" yaml:"server_selection_timeout"`
}

// Client 封装 mongo.Client 与默认数据库名。
type Client struct {
	client          *mongo.Client
	defaultDatabase string
}

// 全局 MongoDB 客户端，通过 Init 初始化、GetClient 获取。
var (
	defaultClientMu sync.RWMutex
	defaultClient   *Client
)

// NewClient 创建并 ping 一个 MongoDB 客户端。
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	client, err := mongo.Connect(cfg.clientOptions())
	if err != nil {
		return nil, fmt.Errorf("连接 MongoDB 失败: %w", err)
	}

	timeout := cfg.ConnectTimeout
	if timeout <= 0 {
		timeout = DefaultConnectTimeout
	}
	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("MongoDB ping 失败: %w", err)
	}

	return &Client{client: client, defaultDatabase: cfg.Database}, nil
}

// Init 初始化全局客户端。重复调用会关闭旧客户端。
func Init(ctx context.Context, cfg Config) error {
	client, err := NewClient(ctx, cfg)
	if err != nil {
		return err
	}

	defaultClientMu.Lock()
	old := defaultClient
	defaultClient = client
	defaultClientMu.Unlock()

	if old != nil {
		_ = old.Close(context.Background())
	}
	return nil
}

// GetClient 获取全局客户端，未初始化时返回 nil。
func GetClient() *Client {
	defaultClientMu.RLock()
	defer defaultClientMu.RUnlock()
	return defaultClient
}

// DB 获取全局客户端的数据库句柄。name 为空时使用配置中的默认数据库。
func DB(name ...string) (*mongo.Database, error) {
	client := GetClient()
	if client == nil {
		return nil, errors.New("mongodb: 未初始化，请先调用 Init")
	}
	return client.Database(name...)
}

// Close 关闭全局客户端并清空引用。
func Close(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	defaultClientMu.Lock()
	client := defaultClient
	defaultClient = nil
	defaultClientMu.Unlock()
	if client == nil {
		return nil
	}
	return client.Close(ctx)
}

// Raw 返回底层 mongo.Client。
func (c *Client) Raw() *mongo.Client {
	if c == nil {
		return nil
	}
	return c.client
}

// DefaultDatabase 返回配置中的默认数据库名。
func (c *Client) DefaultDatabase() string {
	if c == nil {
		return ""
	}
	return c.defaultDatabase
}

// Database 获取数据库句柄。name 为空时使用默认数据库。
func (c *Client) Database(name ...string) (*mongo.Database, error) {
	if c == nil || c.client == nil {
		return nil, errors.New("mongodb: client 未初始化")
	}
	database := c.defaultDatabase
	if len(name) > 0 && name[0] != "" {
		database = name[0]
	}
	if database == "" {
		return nil, errors.New("mongodb: database 名称不能为空")
	}
	return c.client.Database(database), nil
}

// Ping 检查 MongoDB 是否可用。
func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.client == nil {
		return errors.New("mongodb: client 未初始化")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return c.client.Ping(ctx, readpref.Primary())
}

// Close 关闭客户端。
func (c *Client) Close(ctx context.Context) error {
	if c == nil || c.client == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return c.client.Disconnect(ctx)
}

func (c Config) dsn() string {
	if c.URI != "" {
		return c.URI
	}
	host := c.Host
	if host == "" {
		host = defaultHost
	}
	port := c.Port
	if port <= 0 {
		port = defaultPort
	}
	return "mongodb://" + net.JoinHostPort(host, strconv.Itoa(port))
}

func (c Config) clientOptions() *options.ClientOptions {
	opts := options.Client().ApplyURI(c.dsn())

	if c.Username != "" {
		credential := options.Credential{
			Username:    c.Username,
			Password:    c.Password,
			PasswordSet: true,
		}
		if c.AuthSource != "" {
			credential.AuthSource = c.AuthSource
		}
		opts.SetAuth(credential)
	}
	if c.AppName != "" {
		opts.SetAppName(c.AppName)
	}
	if c.MaxPoolSize > 0 {
		opts.SetMaxPoolSize(c.MaxPoolSize)
	}
	if c.MinPoolSize > 0 {
		opts.SetMinPoolSize(c.MinPoolSize)
	}
	if c.ConnectTimeout > 0 {
		opts.SetConnectTimeout(c.ConnectTimeout)
	}
	if c.ServerSelectionTimeout > 0 {
		opts.SetServerSelectionTimeout(c.ServerSelectionTimeout)
	}
	return opts
}
