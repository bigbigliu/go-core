package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/bigbigliu/go-core/database/mongodb"
)

func main() {
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		fmt.Println("set MONGO_URI (for example mongodb://127.0.0.1:27017) to run this example")
		return
	}

	ctx := context.Background()
	client, err := mongodb.NewClient(ctx, mongodb.Config{
		URI:            uri,
		Database:       envString("MONGO_DB", "core"),
		ConnectTimeout: 5 * time.Second,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() { _ = client.Close(context.Background()) }()

	db, err := client.Database()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("connected to database: %s\n", db.Name())

	// 同一个客户端可以访问多个逻辑库。
	logs, err := client.Database("logs")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("another database handle: %s\n", logs.Name())

	if err := client.Ping(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "ping:", err)
		os.Exit(1)
	}
	fmt.Println("ping ok")
}

func envString(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
