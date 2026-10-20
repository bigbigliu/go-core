package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/bigbigliu/go-core/bootstrap"
	"github.com/bigbigliu/go-core/logger"
	"github.com/bigbigliu/go-core/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const devConfigPath = "./config/settings-dev.yml"

func main() {
	if err := run(); err != nil {
		log.Fatalf("服务退出: %v", err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	opts := bootstrap.Options{
		ConfigPath:            configPath(),
		ServiceName:           "go-core-example",
		EnableMemoryRateLimit: true,
		RateLimitRequests:     100,
		RateLimitWindow:       time.Minute,
		Timeout:               5 * time.Second,
	}
	if port := appPort(); port > 0 {
		opts.Port = port
	}

	app, err := bootstrap.New(ctx, opts)
	if err != nil {
		return fmt.Errorf("初始化服务失败: %w", err)
	}
	defer func() { _ = app.Close() }()

	app.Router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, response.OK(gin.H{"status": "ok"}))
	})
	app.Router.GET("/hello/:name", func(c *gin.Context) {
		name := c.Param("name")
		app.Logger.WithOptions(logger.WithContext(c.Request.Context())).
			Info("hello", zap.String("name", name))
		c.JSON(http.StatusOK, response.OK(gin.H{"message": "hello " + name}))
	})

	return app.Run(ctx)
}

// configPath 优先使用 CONFIG 环境变量，其次使用默认配置，
// 最后回退到仓库自带的开发配置。
func configPath() string {
	if path := os.Getenv("CONFIG"); path != "" {
		return path
	}
	if _, err := os.Stat("./config/settings.yml"); err == nil {
		return "./config/settings.yml"
	}
	return devConfigPath
}

func appPort() int {
	value := os.Getenv("APP_PORT")
	if value == "" {
		return 0
	}
	port, err := strconv.Atoi(value)
	if err != nil || port <= 0 {
		return 0
	}
	return port
}
