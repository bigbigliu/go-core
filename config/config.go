package config

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

// Config 表示配置文件的结构体
type Config struct {
	App       *AppConf           `yaml:"app"`       // App app配置
	Db        *DbConf            `yaml:"db"`        // Db 默认数据库配置
	Databases map[string]*DbConf `yaml:"databases"` // Databases 多数据库配置，数量不限
	Redis     *RedisConf         `yaml:"redis"`     // Redis redis配置
	Mongo     *MongoConf         `yaml:"mongo"`     // Mongo mongodb配置
	Logger    *LoggerConf        `yaml:"logger"`    // Logger 日志配置
	Jwt       *JwtConf           `yaml:"jwt"`       // Jwt jwt配置
}

// AppConf app配置
type AppConf struct {
	Host string `yaml:"host"` // Host app启动host
	Port int    `yaml:"port"` // Port app启动port
	Mode string `yaml:"mode"` // Mode gin启动模式
}

// DbConf 数据库配置
type DbConf struct {
	Host               string `yaml:"host"`                 // Host 数据库服务host
	Port               int    `yaml:"port"`                 // Port 数据库服务port
	User               string `yaml:"user"`                 // User 数据库服务用户名
	Password           string `yaml:"password"`             // Password 数据库服务密码
	Name               string `yaml:"name"`                 // Name 数据库名
	MaxIdleConnections int    `yaml:"max_idle_connections"` // MaxIdleConnections 空闲连接池中连接的最大数量
	MaxOpenConnections int    `yaml:"max_open_connections"` // MaxOpenConnections 数据库的最大打开连接数
}

// MongoConf mongodb配置
type MongoConf struct {
	URI                    string `yaml:"uri"`                      // URI 非空时优先使用
	Host                   string `yaml:"host"`                     // Host mongodb服务host
	Port                   int    `yaml:"port"`                     // Port mongodb服务port
	Username               string `yaml:"username"`                 // Username 用户名
	Password               string `yaml:"password"`                 // Password 密码
	AuthSource             string `yaml:"auth_source"`              // AuthSource 认证库
	Database               string `yaml:"database"`                 // Database 默认数据库
	AppName                string `yaml:"app_name"`                 // AppName 应用名
	MaxPoolSize            uint64 `yaml:"max_pool_size"`            // MaxPoolSize 最大连接数
	MinPoolSize            uint64 `yaml:"min_pool_size"`            // MinPoolSize 最小连接数
	ConnectTimeout         int    `yaml:"connect_timeout"`          // ConnectTimeout 连接超时(秒)
	ServerSelectionTimeout int    `yaml:"server_selection_timeout"` // ServerSelectionTimeout 选主超时(秒)
}

// RedisConf redis配置
type RedisConf struct {
	Addr string `yaml:"addr"` // Addr redis服务host
	Port int    `yaml:"port"` // Port redis服务port
	Pwd  string `yaml:"pwd"`  // Pwd redis服务密码
	Db   int    `yaml:"db"`   // Db redis服务数据库
}

// LoggerConf 日志配置
type LoggerConf struct {
	Path  string `yaml:"path"`  // Path 日志保存
	Level string `yaml:"level"` // Level 日志级别
}

// JwtConf jwt配置
type JwtConf struct {
	Secret  string `yaml:"secret"`  // Secret jwt密钥
	Timeout int    `yaml:"timeout"` // Timeout token过期时间 单位 秒/s
}

var (
	configMu        sync.RWMutex
	singletonConfig *Config
)

// DefaultConfigPaths 默认配置文件查找顺序，可通过 CONFIG 环境变量覆盖
var DefaultConfigPaths = []string{
	"./config/settings.yml",
	"./config/settings.yaml",
}

// Load 从指定文件加载配置。加载成功后覆盖全局配置单例。
func Load(filename string) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("读取配置文件 %s 失败: %w", filename, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return fmt.Errorf("配置文件 %s 反序列化失败: %w", filename, err)
	}

	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		return fmt.Errorf("配置文件 %s 校验失败: %w", filename, err)
	}

	configMu.Lock()
	singletonConfig = &cfg
	configMu.Unlock()
	return nil
}

// LoadDefault 按 DefaultConfigPaths 的顺序加载第一个存在的配置文件。
func LoadDefault() error {
	var errs []error
	for _, path := range DefaultConfigPaths {
		if _, err := os.Stat(path); err != nil {
			errs = append(errs, err)
			continue
		}
		return Load(path)
	}
	return fmt.Errorf("未找到可用的配置文件: %w", errors.Join(errs...))
}

// LoadFromEnv 优先读取 CONFIG 环境变量指定的配置文件，否则使用默认查找顺序。
func LoadFromEnv() error {
	if path := os.Getenv("CONFIG"); path != "" {
		return Load(path)
	}
	return LoadDefault()
}

// MustLoad 加载配置，失败时 panic，便于在 main/init 中使用。
func MustLoad(filename string) *Config {
	if err := Load(filename); err != nil {
		panic(err)
	}
	return GetConfig()
}

// MustLoadDefault 按默认顺序加载配置，失败时 panic。
func MustLoadDefault() *Config {
	if err := LoadFromEnv(); err != nil {
		panic(err)
	}
	return GetConfig()
}

// GetConfig 获取全局配置单例。若尚未加载，会尝试按默认顺序加载一次。
// 加载失败时返回 nil，由调用方决定如何处理，库不会主动退出进程。
func GetConfig() *Config {
	configMu.RLock()
	cfg := singletonConfig
	configMu.RUnlock()
	if cfg != nil {
		return cfg
	}

	_ = LoadFromEnv()

	configMu.RLock()
	defer configMu.RUnlock()
	return singletonConfig
}

// SetConfig 设置全局配置单例，便于测试或多环境切换。
// 传入的配置会自动补齐默认值；cfg 为 nil 时忽略。
func SetConfig(cfg *Config) {
	if cfg == nil {
		return
	}
	cfg.applyDefaults()
	configMu.Lock()
	singletonConfig = cfg
	configMu.Unlock()
}

func (c *Config) applyDefaults() {
	if c.App == nil {
		c.App = &AppConf{}
	}
	if c.Db == nil {
		c.Db = &DbConf{}
	}
	c.Db.applyDefaults()
	for name, db := range c.Databases {
		if db == nil {
			db = &DbConf{}
			c.Databases[name] = db
		}
		db.applyDefaults()
	}
	if c.Redis == nil {
		c.Redis = &RedisConf{}
	}
	if c.Mongo == nil {
		c.Mongo = &MongoConf{}
	}
	if c.Mongo.Host == "" {
		c.Mongo.Host = "127.0.0.1"
	}
	if c.Mongo.Port == 0 {
		c.Mongo.Port = 27017
	}
	if c.Mongo.ConnectTimeout <= 0 {
		c.Mongo.ConnectTimeout = 10
	}
	if c.Mongo.ServerSelectionTimeout <= 0 {
		c.Mongo.ServerSelectionTimeout = 10
	}
	if c.Logger == nil {
		c.Logger = &LoggerConf{}
	}
	if c.Jwt == nil {
		c.Jwt = &JwtConf{}
	}

	if c.Redis.Port == 0 {
		c.Redis.Port = 6379
	}
	if c.Logger.Path == "" {
		c.Logger.Path = "./log"
	}
	if c.Logger.Level == "" {
		c.Logger.Level = "info"
	}
	if c.Jwt.Timeout <= 0 {
		c.Jwt.Timeout = 7200
	}
}

func (d *DbConf) applyDefaults() {
	if d.Port == 0 {
		d.Port = 3306
	}
	if d.MaxIdleConnections <= 0 {
		d.MaxIdleConnections = 10
	}
	if d.MaxOpenConnections <= 0 {
		d.MaxOpenConnections = 100
	}
}

func (c *Config) validate() error {
	if c.Db.Port < 0 || c.Redis.Port < 0 || c.Mongo.Port < 0 || c.App.Port < 0 {
		return errors.New("端口不能为负数")
	}
	for name, db := range c.Databases {
		if db.Port < 0 {
			return fmt.Errorf("databases.%s.port 不能为负数", name)
		}
	}
	return nil
}
