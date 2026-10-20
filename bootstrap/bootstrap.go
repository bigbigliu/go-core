// Package bootstrap wires go-core components together for a web service.
//
// It loads configuration, initializes logging, optionally opens MySQL, Redis
// and MongoDB connections, and builds a Gin engine with the standard middleware
// stack. Services can add routes and call Run instead of repeating setup code.
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/bigbigliu/go-core/config"
	"github.com/bigbigliu/go-core/database/mongodb"
	"github.com/bigbigliu/go-core/database/mysql"
	"github.com/bigbigliu/go-core/database/redis"
	"github.com/bigbigliu/go-core/logger"
	"github.com/bigbigliu/go-core/middleware"
	"github.com/gin-gonic/gin"
	goRedis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Options controls which components bootstrap initializes.
type Options struct {
	// ConfigPath is the YAML config path. When empty, config.LoadFromEnv is used.
	ConfigPath string
	// Config can be supplied directly; it takes precedence over ConfigPath.
	Config *config.Config

	// Host/Port override config.App.Host and config.App.Port.
	Host string
	Port int

	ServiceName       string
	DisableLogger     bool
	DisableConsoleLog bool

	InitMySQL bool
	InitRedis bool
	InitMongo bool

	// Timeout is the request timeout middleware duration. Defaults to 5s.
	Timeout time.Duration
	// EnableMemoryRateLimit enables the per-IP in-memory rate limiter.
	EnableMemoryRateLimit bool
	RateLimitRequests     int
	RateLimitWindow       time.Duration
	// SkipLogPaths are access-log paths to skip. Defaults to /health and /metrics.
	SkipLogPaths []string
}

// App holds initialized components and the Gin engine.
type App struct {
	Config *config.Config
	Logger *zap.Logger
	MySQL  *mysql.Manager
	Redis  *goRedis.Client
	Mongo  *mongodb.Client
	Router *gin.Engine

	limiter *middleware.MemoryLimiter
}

// New initializes the requested components. On error, already initialized
// resources are closed before returning.
func New(ctx context.Context, opts Options) (*App, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	cfg, err := loadConfig(opts)
	if err != nil {
		return nil, err
	}
	applyOverrides(cfg, opts)

	app := &App{Config: cfg}

	if !opts.DisableLogger {
		consoleOutput := !opts.DisableConsoleLog && cfg.App.Mode != "release"
		if err := logger.InitializeLogger(&logger.CoreLog{
			LogDir:        cfg.Logger.Path,
			LogLevel:      cfg.Logger.Level,
			ConsoleOutPut: consoleOutput,
			ServiceName:   serviceName(opts),
		}); err != nil {
			return nil, fmt.Errorf("初始化日志失败: %w", err)
		}
		app.Logger = logger.Logger
	}

	if opts.InitMySQL {
		if err := app.initMySQL(); err != nil {
			_ = app.Close()
			return nil, err
		}
	}
	if opts.InitRedis {
		if err := app.initRedis(); err != nil {
			_ = app.Close()
			return nil, err
		}
	}
	if opts.InitMongo {
		if err := app.initMongo(ctx); err != nil {
			_ = app.Close()
			return nil, err
		}
	}

	app.Router = app.buildRouter(opts)
	return app, nil
}

// Gin returns the initialized Gin engine.
func (a *App) Gin() *gin.Engine {
	return a.Router
}

// Run starts the HTTP server and shuts it down when ctx is canceled.
func (a *App) Run(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	host := a.Config.App.Host
	if host == "" {
		host = "0.0.0.0"
	}
	port := a.Config.App.Port
	if port <= 0 {
		port = 8080
	}
	addr := net.JoinHostPort(host, strconv.Itoa(port))

	server := &http.Server{
		Addr:              addr,
		Handler:           a.Router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		if a.Logger != nil {
			a.Logger.Info("HTTP 服务启动", zap.String("addr", addr))
		}
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("监听 %s 失败: %w", addr, err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	if a.Logger != nil {
		a.Logger.Info("正在关闭 HTTP 服务")
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}

// Close closes initialized databases, caches and flushes logs.
func (a *App) Close() error {
	var errs []error
	if a.limiter != nil {
		a.limiter.Close()
	}
	if a.MySQL != nil {
		errs = append(errs, a.MySQL.CloseAll())
	}
	if a.Mongo != nil {
		errs = append(errs, a.Mongo.Close(context.Background()))
	}
	if a.Redis != nil {
		errs = append(errs, a.Redis.Close())
		redis.Redisclient = nil
		a.Redis = nil
	}
	if a.Logger != nil {
		errs = append(errs, logger.Sync())
	}
	return errors.Join(errs...)
}

func (a *App) initMySQL() error {
	cfg := a.Config
	manager := mysql.NewManager()

	if cfg.Db != nil && cfg.Db.Name != "" {
		if _, err := manager.Open("default", mysqlConfig(cfg.Db)); err != nil {
			_ = manager.CloseAll()
			return fmt.Errorf("初始化 MySQL default 连接失败: %w", err)
		}
	}
	for name, dbCfg := range cfg.Databases {
		if dbCfg == nil || dbCfg.Name == "" {
			continue
		}
		if _, err := manager.Open(name, mysqlConfig(dbCfg)); err != nil {
			_ = manager.CloseAll()
			return fmt.Errorf("初始化 MySQL %s 连接失败: %w", name, err)
		}
	}
	if manager.Len() == 0 {
		return errors.New("mysql 已启用，但未配置 db.name 或 databases")
	}
	a.MySQL = manager
	return nil
}

func (a *App) initRedis() error {
	cfg := a.Config
	if cfg.Redis == nil || cfg.Redis.Addr == "" {
		return errors.New("redis 已启用，但未配置 redis.addr")
	}
	if err := (&redis.InitRedisReq{
		Addr: cfg.Redis.Addr,
		Port: cfg.Redis.Port,
		Pwd:  cfg.Redis.Pwd,
		Db:   cfg.Redis.Db,
	}).InitRedis(); err != nil {
		return err
	}
	a.Redis = redis.Redisclient
	return nil
}

func (a *App) initMongo(ctx context.Context) error {
	cfg := a.Config
	if cfg.Mongo == nil {
		return errors.New("mongodb 已启用，但未配置 mongo")
	}

	client, err := mongodb.NewClient(ctx, mongodb.Config{
		URI:                    cfg.Mongo.URI,
		Host:                   cfg.Mongo.Host,
		Port:                   cfg.Mongo.Port,
		Database:               cfg.Mongo.Database,
		Username:               cfg.Mongo.Username,
		Password:               cfg.Mongo.Password,
		AuthSource:             cfg.Mongo.AuthSource,
		AppName:                cfg.Mongo.AppName,
		MaxPoolSize:            cfg.Mongo.MaxPoolSize,
		MinPoolSize:            cfg.Mongo.MinPoolSize,
		ConnectTimeout:         time.Duration(cfg.Mongo.ConnectTimeout) * time.Second,
		ServerSelectionTimeout: time.Duration(cfg.Mongo.ServerSelectionTimeout) * time.Second,
	})
	if err != nil {
		return err
	}
	a.Mongo = client
	return nil
}

func (a *App) buildRouter(opts Options) *gin.Engine {
	cfg := a.Config
	if cfg.App.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	skipPaths := opts.SkipLogPaths
	if skipPaths == nil {
		skipPaths = []string{"/health", "/metrics"}
	}

	router := gin.New()
	router.Use(
		gin.Recovery(),
		middleware.RequestIDMiddleware(),
		middleware.CorsMiddleware(),
		middleware.GinLogger(skipPaths, nil),
	)

	if opts.EnableMemoryRateLimit {
		requests := opts.RateLimitRequests
		if requests <= 0 {
			requests = 100
		}
		window := opts.RateLimitWindow
		if window <= 0 {
			window = time.Minute
		}
		a.limiter = middleware.NewMemoryLimiter(requests, window)
		router.Use(a.limiter.Middleware())
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	router.Use(middleware.TimeoutMiddleware(timeout))
	return router
}

func loadConfig(opts Options) (*config.Config, error) {
	if opts.Config != nil {
		config.SetConfig(opts.Config)
		return opts.Config, nil
	}

	if opts.ConfigPath != "" {
		if err := config.Load(opts.ConfigPath); err != nil {
			return nil, err
		}
	} else if err := config.LoadFromEnv(); err != nil {
		return nil, err
	}

	cfg := config.GetConfig()
	if cfg == nil {
		return nil, errors.New("配置未加载")
	}
	return cfg, nil
}

func applyOverrides(cfg *config.Config, opts Options) {
	if opts.Host != "" {
		cfg.App.Host = opts.Host
	}
	if opts.Port > 0 {
		cfg.App.Port = opts.Port
	}
}

func mysqlConfig(db *config.DbConf) *mysql.Config {
	return &mysql.Config{
		DbHost:      db.Host,
		DbPort:      db.Port,
		DbUser:      db.User,
		DbPwd:       db.Password,
		DbName:      db.Name,
		MaxIdleConn: db.MaxIdleConnections,
		MaxOpenConn: db.MaxOpenConnections,
	}
}

func serviceName(opts Options) string {
	if opts.ServiceName != "" {
		return opts.ServiceName
	}
	return "go-core-service"
}
