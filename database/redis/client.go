package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/bigbigliu/go-core/logger"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Redisclient 全局redis客户端
var Redisclient *redis.Client

// InitRedisReq 请求参数
type InitRedisReq struct {
	Addr string `json:"addr"` // Addr redis服务host
	Port int    `json:"port"` // Port redis服务port
	Pwd  string `json:"pwd"`  // Pwd redis服务密码
	Db   int    `json:"db"`   // Db redis服务数据库
}

// InitRedis 初始化redis连接。失败时返回错误，由调用方决定是否退出。
func (h *InitRedisReq) InitRedis() error {
	logInfo("Redis", zap.String("conn", "connecting..."))

	addr := h.Addr + ":" + strconv.Itoa(h.Port)
	logInfo("Redis", zap.String("redis_addr", addr))

	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     h.Pwd,
		DB:           h.Db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return fmt.Errorf("redis 连接失败: %w", err)
	}

	Redisclient = client
	logInfo("Redis", zap.String("conn", "Redis连接成功"))
	return nil
}

func logInfo(msg string, fields ...zap.Field) {
	if logger.Logger != nil {
		logger.Logger.Info(msg, fields...)
	}
}
