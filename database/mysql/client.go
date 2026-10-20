package mysql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bigbigliu/go-core/logger"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormLog "gorm.io/gorm/logger"
)

// DBClient 全局DB客户端
var DBClient *gorm.DB

// GenerateDSNParam 生成dsn参数
type GenerateDSNParam struct {
	DbHost          string        `json:"db_host"`           // DbHost 数据库服务host
	DbPort          int           `json:"db_port"`           // DbPort 数据库服务port
	DbUser          string        `json:"db_user"`           // DbUser 数据库服务用户名
	DbPwd           string        `json:"db_pwd"`            // DbPwd 数据库服务密码
	DbName          string        `json:"db_name"`           // DbName 数据库名
	MaxIdleConn     int           `json:"max_idle_conn"`     // MaxIdleConn 空闲连接数, <=0 时使用默认值
	MaxOpenConn     int           `json:"max_open_conn"`     // MaxOpenConn 最大打开连接数, <=0 时使用默认值
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"` // ConnMaxLifetime 连接最大存活时间, <=0 时使用 1 小时
}

// Config 是 GenerateDSNParam 的别名，便于在连接多个数据库时使用。
type Config = GenerateDSNParam

// Open 创建一个新的数据库连接，不修改全局 DBClient。
// 适用于同时连接多个数据库的场景。
func (h *GenerateDSNParam) Open() (*gorm.DB, error) {
	if h.DbName == "" {
		return nil, errors.New("database name 不能为空")
	}

	logInfo("DB", zap.String("conn", "connecting..."))

	gormConfig := &gorm.Config{}
	if logger.Logger != nil {
		gormConfig.Logger = logger.NewCustomLogger(logger.Logger, context.Background(), gormLog.Info)
	}

	db, err := gorm.Open(mysql.Open(h.GenerateDSN()), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库连接池失败: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("数据库 ping 失败: %w", err)
	}

	maxIdle := h.MaxIdleConn
	if maxIdle <= 0 {
		maxIdle = 10
	}
	maxOpen := h.MaxOpenConn
	if maxOpen <= 0 {
		maxOpen = 100
	}
	connMaxLifetime := h.ConnMaxLifetime
	if connMaxLifetime <= 0 {
		connMaxLifetime = time.Hour
	}
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)

	logInfo("DB", zap.String("conn", "数据库连接成功"))
	return db, nil
}

// InitDB 初始化全局 DBClient。失败时返回错误，由调用方决定是否退出。
// 需要连接多个数据库时请使用 Manager 或 GenerateDSNParam.Open。
func (h *GenerateDSNParam) InitDB() error {
	db, err := h.Open()
	if err != nil {
		return err
	}
	DBClient = db
	return nil
}

// GenerateDSN 生成 dsn。注意该字符串包含密码，不要直接输出到日志。
func (h *GenerateDSNParam) GenerateDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		h.DbUser, h.DbPwd, h.DbHost, h.DbPort, h.DbName)
}

// SafeDSN 生成用于日志输出的脱敏 dsn。
func (h *GenerateDSNParam) SafeDSN() string {
	return fmt.Sprintf("%s:***@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		h.DbUser, h.DbHost, h.DbPort, h.DbName)
}

func logInfo(msg string, fields ...zap.Field) {
	if logger.Logger != nil {
		logger.Logger.Info(msg, fields...)
	}
}
