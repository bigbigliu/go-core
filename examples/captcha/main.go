package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/bigbigliu/go-core/captcha"
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

	store := captcha.NewRedisStore(redis.Redisclient, time.Minute)
	if err := store.Set("demo-id", "123456"); err != nil {
		fmt.Fprintln(os.Stderr, "set:", err)
		os.Exit(1)
	}

	fmt.Println("verify correct answer:", store.Verify("demo-id", "123456", false))
	fmt.Println("verify wrong answer:", store.Verify("demo-id", "000000", false))

	// 清除验证码
	_ = store.Get("demo-id", true)
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
