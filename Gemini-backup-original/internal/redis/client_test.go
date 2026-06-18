package redis

import (
	"testing"
)

func TestInit_NoRedis(t *testing.T) {
	// Without REDIS_ADDR or REDIS_URL pointing to a real server,
	// Init should fail with a connection error.
	// We don't have a Redis server in CI, so just verify it doesn't panic.
	// In a real scenario, you'd use miniredis or testcontainers.
	t.Skip("skipping: requires running Redis server")
}

func TestClose_NilClient(t *testing.T) {
	// Client is nil by default (without Init), Close should not panic
	Client = nil
	Close() // should not panic
}

func TestOpTimeout(t *testing.T) {
	if OpTimeout == 0 {
		t.Error("OpTimeout should be non-zero")
	}
}
