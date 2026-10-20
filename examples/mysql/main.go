package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/bigbigliu/go-core/database/mysql"
)

func main() {
	host := os.Getenv("MYSQL_HOST")
	if host == "" {
		fmt.Println("set MYSQL_HOST/MYSQL_USER/MYSQL_PASSWORD/MYSQL_DB to run this example")
		fmt.Println("optional: MYSQL_PORT (default 3306), MYSQL_ORDERS_DB (second connection)")
		return
	}

	manager := mysql.NewManager()
	defer func() { _ = manager.CloseAll() }()

	primary, err := manager.Open("primary", &mysql.Config{
		DbHost:      host,
		DbPort:      envInt("MYSQL_PORT", 3306),
		DbUser:      envString("MYSQL_USER", "root"),
		DbPwd:       envString("MYSQL_PASSWORD", ""),
		DbName:      envString("MYSQL_DB", "core"),
		MaxIdleConn: 10,
		MaxOpenConn: 100,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "open primary:", err)
		os.Exit(1)
	}
	fmt.Println("connected primary")

	// 同一个 Manager 可以注册任意多个命名连接。
	if ordersDB := os.Getenv("MYSQL_ORDERS_DB"); ordersDB != "" {
		if _, err := manager.Open("orders", &mysql.Config{
			DbHost: host,
			DbPort: envInt("MYSQL_PORT", 3306),
			DbUser: envString("MYSQL_USER", "root"),
			DbPwd:  envString("MYSQL_PASSWORD", ""),
			DbName: ordersDB,
		}); err != nil {
			fmt.Fprintln(os.Stderr, "open orders:", err)
			os.Exit(1)
		}
		fmt.Println("connected orders")
	}

	sqlDB, err := primary.DB()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("registered connections: %v, open connections: %d\n", manager.Names(), sqlDB.Stats().OpenConnections)
}

func envString(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
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
