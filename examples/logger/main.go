package main

import (
	"context"
	"fmt"
	"os"

	"github.com/bigbigliu/go-core/logger"
	"go.uber.org/zap"
)

func main() {
	dir, err := os.MkdirTemp("", "go-core-log-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	if err := logger.InitializeLogger(&logger.CoreLog{
		LogDir:        dir,
		LogLevel:      "debug",
		ConsoleOutPut: true,
		ServiceName:   "logger-example",
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() { _ = logger.Sync() }()

	logger.Logger.Info("hello from logger example", zap.String("component", "logger"))

	ctx := logger.ContextWithRequestID(context.Background(), "req-example-1")
	logger.Logger.WithOptions(logger.WithContext(ctx)).
		Info("this log line carries a request id")

	data, err := os.ReadFile(dir + "/app.log")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("log file:\n%s", data)
}
