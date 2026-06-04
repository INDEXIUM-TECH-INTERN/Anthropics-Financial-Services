package redis

import (
	"context"
	"fmt"
	"os"

	goredis "github.com/redis/go-redis/v9"
)

var Client *goredis.Client
var Ctx = context.Background()

func Init() error {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	password := os.Getenv("REDIS_PASSWORD")
	db := 0 // default

	Client = goredis.NewClient(&goredis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

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
