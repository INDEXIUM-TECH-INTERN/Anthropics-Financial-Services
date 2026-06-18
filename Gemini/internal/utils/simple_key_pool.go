package utils

import (
	"errors"
	"math/rand"
	"strings"
	"sync"
	"time"
)

// SimpleKeyPool quản lý một pool các API keys đơn giản
type SimpleKeyPool struct {
	keys     []string
	index    int
	mu       sync.RWMutex
	lastUsed map[string]time.Time
}

// NewSimpleKeyPool tạo một SimpleKeyPool từ chuỗi keys
func NewSimpleKeyPool(keysString string) (*SimpleKeyPool, error) {
	keys := strings.Split(keysString, ",")

	// Validate
	if len(keys) == 0 {
		return nil, errors.New("no keys provided")
	}

	// Remove empty keys
	validKeys := make([]string, 0)
	for _, key := range keys {
		if strings.TrimSpace(key) != "" {
			validKeys = append(validKeys, strings.TrimSpace(key))
		}
	}

	if len(validKeys) == 0 {
		return nil, errors.New("no valid keys provided")
	}

	// Shuffle để không luôn dùng key đầu tiên
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(validKeys), func(i, j int) {
		validKeys[i], validKeys[j] = validKeys[j], validKeys[i]
	})

	return &SimpleKeyPool{
		keys:     validKeys,
		lastUsed: make(map[string]time.Time),
	}, nil
}

// GetKey trả về một key từ pool (round-robin)
func (kp *SimpleKeyPool) GetKey() string {
	kp.mu.Lock()
	defer kp.mu.Unlock()

	key := kp.keys[kp.index]
	kp.lastUsed[key] = time.Now()
	kp.index = (kp.index + 1) % len(kp.keys)

	return key
}

// GetRandomKey trả về một ngẫu nhiên key từ pool
func (kp *SimpleKeyPool) GetRandomKey() string {
	kp.mu.Lock()
	defer kp.mu.Unlock()

	index := rand.Intn(len(kp.keys))
	key := kp.keys[index]
	kp.lastUsed[key] = time.Now()

	return key
}

// GetAllKeys trả về tất cả keys
func (kp *SimpleKeyPool) GetAllKeys() []string {
	kp.mu.RLock()
	defer kp.mu.RUnlock()

	// Copy slice để tránh race condition
	keys := make([]string, len(kp.keys))
	copy(keys, kp.keys)
	return keys
}

// GetKeysCount trả về số lượng keys hợp lệ
func (kp *SimpleKeyPool) GetKeysCount() int {
	kp.mu.RLock()
	defer kp.mu.RUnlock()
	return len(kp.keys)
}

// IsEmpty kiểm tra pool có rỗng không
func (kp *SimpleKeyPool) IsEmpty() bool {
	kp.mu.RLock()
	defer kp.mu.RUnlock()
	return len(kp.keys) == 0
}

// Stats trả về thống kê usage
func (kp *SimpleKeyPool) Stats() map[string]time.Time {
	kp.mu.RLock()
	defer kp.mu.RUnlock()

	// Copy map để tránh race condition
	stats := make(map[string]time.Time)
	for k, v := range kp.lastUsed {
		stats[k] = v
	}
	return stats
}