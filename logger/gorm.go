package logger

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"go.uber.org/zap"
	gormLog "gorm.io/gorm/logger"
)

// CustomLogger gorm日志记录器
type CustomLogger struct {
	logger  *zap.Logger
	context context.Context
	level   gormLog.LogLevel
}

// NewCustomLogger gorm日志记录器
func NewCustomLogger(zapLogger *zap.Logger, ctx context.Context, level gormLog.LogLevel) *CustomLogger {
	return &CustomLogger{
		logger:  zapLogger,
		context: ctx,
		level:   level, // 设置日志级别
	}
}

// LogMode 设置日志记录模式
func (l *CustomLogger) LogMode(level gormLog.LogLevel) gormLog.Interface {
	l.level = level
	return l
}

// Info 记录信息日志
func (l *CustomLogger) Info(ctx context.Context, s string, args ...interface{}) {
	if l.logger == nil || l.level < gormLog.Info {
		return
	}
	l.logger.Info(fmt.Sprintf(s, args...), zap.Any(RequestIDKey, requestID(ctx)))
}

// Warn 记录警告日志
func (l *CustomLogger) Warn(ctx context.Context, s string, args ...interface{}) {
	if l.logger == nil || l.level < gormLog.Warn {
		return
	}
	l.logger.Warn(fmt.Sprintf(s, args...), zap.Any(RequestIDKey, requestID(ctx)))
}

// Error 记录错误日志
func (l *CustomLogger) Error(ctx context.Context, s string, args ...interface{}) {
	if l.logger == nil || l.level < gormLog.Error {
		return
	}
	l.logger.Error(fmt.Sprintf(s, args...), zap.Any(RequestIDKey, requestID(ctx)))
}

// Trace 记录追踪日志
func (l *CustomLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.logger == nil || l.level == gormLog.Silent {
		return
	}

	funcName, funcline := getCallingFunction()

	s, ts := fc()
	fields := []zap.Field{
		zap.Any(RequestIDKey, requestID(ctx)),      // requestID
		zap.String("sql", s),                       // sql语句
		zap.Int64("rows", ts),                      // 受影响行数
		zap.String("function", funcName),           // 记录执行的 SQL 函数
		zap.Int("function_line", funcline),         // 记录执行的 SQL 函数行号
		zap.Duration("elapsed", time.Since(begin)), // sql耗时
	}
	if err != nil {
		if err.Error() == "record not found" {
			l.logger.Warn("SQL Warn", append(fields, zap.Error(err))...)
		} else {
			l.logger.Error("SQL Error", append(fields, zap.Error(err))...)
		}
	} else {
		l.logger.Info("SQL Query", fields...)
	}
}

func requestID(ctx context.Context) any {
	id, ok := RequestIDFromContext(ctx)
	if !ok {
		return nil
	}
	return id
}

// getCallingFunction 获取当前调用的函数名
func getCallingFunction() (string, int) {
	pc := make([]uintptr, 10)
	runtime.Callers(5, pc)
	frames := runtime.CallersFrames(pc)
	frame, _ := frames.Next()
	return frame.Function, frame.Line
}
