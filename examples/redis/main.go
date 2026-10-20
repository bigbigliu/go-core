package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/bigbigliu/go-core/database/redis"
)

func main() {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		fmt.Println("set REDIS_ADDR (for example 127.0.0.1:6379) to run this example")
		return
	}
	host, portText, err := net.SplitHostPort(addr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid REDIS_ADDR:", err)
		os.Exit(1)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid REDIS_ADDR port:", err)
		os.Exit(1)
	}

	if err := (&redis.InitRedisReq{
		Addr: host,
		Port: port,
		Pwd:  os.Getenv("REDIS_PASSWORD"),
		Db:   envInt("REDIS_DB", 0),
	}).InitRedis(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ctx := context.Background()
	if err := redis.Redisclient.Set(ctx, "go-core:example", "hello", 0).Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	value, err := redis.Redisclient.Get(ctx, "go-core:example").Result()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("redis value: %s\n", value)
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return n
}
