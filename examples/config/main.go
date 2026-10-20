package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/bigbigliu/go-core/config"
)

func main() {
	path := os.Getenv("CONFIG")
	if path == "" {
		path = "./config/settings-dev.yml"
	}
	if err := config.Load(path); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	cfg := config.GetConfig()
	fmt.Printf("app: %s:%d (%s)\n", cfg.App.Host, cfg.App.Port, cfg.App.Mode)
	fmt.Printf("default db: %s@%s:%d/%s\n", cfg.Db.User, cfg.Db.Host, cfg.Db.Port, cfg.Db.Name)

	names := make([]string, 0, len(cfg.Databases))
	for name := range cfg.Databases {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		db := cfg.Databases[name]
		fmt.Printf("db[%s]: %s@%s:%d/%s (idle=%d, open=%d)\n",
			name, db.User, db.Host, db.Port, db.Name, db.MaxIdleConnections, db.MaxOpenConnections)
	}

	fmt.Printf("mongo: %s:%d/%s\n", cfg.Mongo.Host, cfg.Mongo.Port, cfg.Mongo.Database)
}
