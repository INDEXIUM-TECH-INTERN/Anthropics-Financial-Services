package redis

import (
	"context"
	"fmt"
	"os"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// OpTimeout cho các Redis operations (Ping, Get, Set, v.v.)
const OpTimeout = 5 * time.Second

var Client *goredis.Client
var Ctx = context.Background()

func Init() error {
	addr := os.Getenv("REDIS_ADDR")
	password := os.Getenv("REDIS_PASSWORD")
	db := 0 // default

	var opts *goredis.Options
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		parsed, err := goredis.ParseURL(redisURL)
		if err != nil {
			return fmt.Errorf("invalid REDIS_URL: %w", err)
		}
		opts = parsed
		addr = parsed.Addr
	} else {
		if addr == "" {
			addr = "127.0.0.1:6379"
		}
		opts = &goredis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		}
	}

	Client = goredis.NewClient(opts)

	_, err := Client.Ping(Ctx).Result()
	if err != nil {
		Client.Close()
		Client = nil
		return fmt.Errorf("failed to connect to Redis at %s: %w", addr, err)
	}

	fmt.Printf("✅ [Redis] Connected to %s\n", addr)
	return nil
}

func Close() {
	if Client != nil {
		Client.Close()
	}
}
