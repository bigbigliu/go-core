package main

import (
	"context"

	"github.com/bigbigliu/go-core/config"
	"github.com/bigbigliu/go-core/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func init() {
	// 初始化日志记录器
	coreLOG := &logger.CoreLog{
		LogDir:   config.GetConfig().Logger.Path,
		LogLevel: config.GetConfig().Logger.Level,
	}
	logger.InitializeLogger(coreLOG)

	// TODO 初始化redis连接

	// TODO 初始化数据库连接

	// TODO 初始化Api接口
	if config.GetConfig().App.Mode == "info" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}
}

func main() {
	logger.Logger.Info("Go-core Start Successfully", zap.String("X-Request-ID", "Program unique ID"))                                  // 旧
	logger.Logger.WithOptions(logger.WithContext(context.Background())).Info("Go-core Start Successfully", zap.String("msg", "gin请求")) // 新
}
