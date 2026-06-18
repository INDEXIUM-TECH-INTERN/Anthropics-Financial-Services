package utils

import (
	"errors"
	"math/rand"
	"strings"
	"sync"
	"time"
)

// KeyPool quản lý một pool các API keys với rotation tự động
type KeyPool struct {
	keys       []string
	index      int
	mu         sync.RWMutex
	lastUsed   map[string]time.Time
}

// NewKeyPool tạo một KeyPool từ chuỗi keys được phân cách bởi dấu phẩy
func NewKeyPool(keysString string) (*KeyPool, error) {
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

	// Random shuffle để không luôn dùng key đầu tiên
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(validKeys), func(i, j int) {
		validKeys[i], validKeys[j] = validKeys[j], validKeys[i]
	})

	return &KeyPool{
		keys:     validKeys,
		lastUsed: make(map[string]time.Time),
	}, nil
}

// GetKey trả về một key từ pool (round-robin)
func (kp *KeyPool) GetKey() string {
	kp.mu.Lock()
	defer kp.mu.Unlock()

	key := kp.keys[kp.index]
	kp.lastUsed[key] = time.Now()
	kp.index = (kp.index + 1) % len(kp.keys)

	return key
}

// GetRandomKey trả về một ngẫu nhiên key từ pool
func (kp *KeyPool) GetRandomKey() string {
	kp.mu.Lock()
	defer kp.mu.Unlock()

	index := rand.Intn(len(kp.keys))
	key := kp.keys[index]
	kp.lastUsed[key] = time.Now()

	return key
}

// GetLeastUsedKey trả về key ít được dùng nhất
func (kp *KeyPool) GetLeastUsedKey() string {
	kp.mu.RLock()
	defer kp.mu.RUnlock()

	if len(kp.keys) == 0 {
		return ""
	}

	var leastUsedKey string
	var earliestTime time.Time

	for i, key := range kp.keys {
		lastUsed, exists := kp.lastUsed[key]
		if !exists {
			// Key chưa từng được dùng, return ngay
			return key
		}

		if i == 0 || lastUsed.Before(earliestTime) {
			leastUsedKey = key
			earliestTime = lastUsed
		}
	}

	return leastUsedKey
}

// GetKeysCount trả về số lượng keys hợp lệ trong pool
func (kp *KeyPool) GetKeysCount() int {
	kp.mu.RLock()
	defer kp.mu.RUnlock()
	return len(kp.keys)
}

// GetAllKeys trả về tất cả keys (để debugging)
func (kp *KeyPool) GetAllKeys() []string {
	kp.mu.RLock()
	defer kp.mu.RUnlock()

	// Copy slice để tránh race condition
	keys := make([]string, len(kp.keys))
	copy(keys, kp.keys)
	return keys
}

// IsEmpty kiểm tra pool có rỗng không
func (kp *KeyPool) IsEmpty() bool {
	kp.mu.RLock()
	defer kp.mu.RUnlock()
	return len(kp.keys) == 0
}

// Stats trả về thống kê usage
func (kp *KeyPool) Stats() map[string]time.Time {
	kp.mu.RLock()
	defer kp.mu.RUnlock()

	// Copy map để tránh race condition
	stats := make(map[string]time.Time)
	for k, v := range kp.lastUsed {
		stats[k] = v
	}
	return stats
}

// ResetUsage reset usage statistics
func (kp *KeyPool) ResetUsage() {
	kp.mu.Lock()
	defer kp.mu.Unlock()
	kp.lastUsed = make(map[string]time.Time)
}

// AddKey thêm một key mới vào pool
func (kp *KeyPool) AddKey(key string) {
	kp.mu.Lock()
	defer kp.mu.Unlock()

	// Check if key already exists
	for _, existingKey := range kp.keys {
		if existingKey == key {
			return
		}
	}

	kp.keys = append(kp.keys, key)
}

// RemoveKey xóa một key khỏi pool
func (kp *KeyPool) RemoveKey(key string) {
	kp.mu.Lock()
	defer kp.mu.Unlock()

	for i, existingKey := range kp.keys {
		if existingKey == key {
			kp.keys = append(kp.keys[:i], kp.keys[i+1:]...)
			delete(kp.lastUsed, key)
			return
		}
	}
}

// RotateKeys xáo trộn ngẫu nhiên tất cả keys
func (kp *KeyPool) RotateKeys() {
	kp.mu.Lock()
	defer kp.mu.Unlock()

	rand.Shuffle(len(kp.keys), func(i, j int) {
		kp.keys[i], kp.keys[j] = kp.keys[j], kp.keys[i]
	})
}

// HealthScore trả về health score dựa trên usage pattern
func (kp *KeyPool) HealthScore() float64 {
	kp.mu.RLock()
	defer kp.mu.RUnlock()

	if len(kp.keys) == 0 {
		return 0.0
	}

	now := time.Now()

	// Tính variance trong usage
	var variance float64
	if len(kp.lastUsed) > 0 {
		var sum time.Duration
		var count int

		for _, t := range kp.lastUsed {
			if !t.IsZero() {
				sum += now.Sub(t)
				count++
			}
		}

		if count > 0 {
			avg := sum / time.Duration(count)

			for _, t := range kp.lastUsed {
				if !t.IsZero() {
					diff := now.Sub(t) - avg
					variance += float64(diff) * float64(diff)
				}
			}

			variance = variance / float64(count)
		}
	}

	// Score từ 0-1, cao hơn là tốt
	score := 1.0 - min(1.0, variance/float64(time.Hour*24)) // Normalize by 24 hours
	return score
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}