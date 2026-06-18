package core

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// KeyPool quản lý pool của Gemini API keys với rotation tự động
type KeyPool struct {
	Keys          []string
	Weights       []float64  // weight cho mỗi key để load balancing
	CurrentIndex int
	HealthStatus  []bool     // track health status của từng key
	TotalUsage    []int64    // total usage count cho từng key
	TotalFailures []int      // total failure count cho từng key
	KeysInUse     map[string]int // count in-flight requests per key
	LastUsed     []time.Time // last used time cho từng key
	LastRotate    time.Time
	mu            sync.RWMutex

	// Config
	MaxKeys       int
	RotationInterval time.Duration
	FailoverThreshold int // số key failure trước khi rotate
}

// KeyStats chứa thông tin về performance của từng key
type KeyStats struct {
	Key          string  `json:"key"`
	Weight       float64 `json:"weight"`
	SuccessRate  float64 `json:"success_rate"`
	UsageCount   int64   `json:"usage_count"`
	LastUsed     time.Time `json:"last_used"`
	LatencyAvg   float64 `json:"latency_avg"`
	LatencyMin   float64 `json:"latency_min"`
	LatencyMax   float64 `json:"latency_max"`
}

// NewKeyPool khởi tạo key pool từ file .env
func NewKeyPool() *KeyPool {
	keys := loadKeysFromEnv()

	pool := &KeyPool{
		Keys:          keys,
		Weights:       make([]float64, len(keys)),
		HealthStatus:  make([]bool, len(keys)),
		TotalUsage:    make([]int64, len(keys)),
		KeysInUse:     make(map[string]int),
		MaxKeys:       len(keys),
		RotationInterval: 5 * time.Minute, // rotate every 5 minutes
		FailoverThreshold: 2, // rotate after 2 consecutive failures per key
	}

	// Initialize Weights evenly
	for i := range pool.Weights {
		pool.Weights[i] = 1.0 / float64(len(keys))
	}

	// Mark all keys as healthy initially
	for i := range pool.HealthStatus {
		pool.HealthStatus[i] = true
	}

	// Start rotation scheduler
	go pool.startRotationScheduler()

	return pool
}

// GetNextKey lấy key tiếp theo với load balancing
func (kp *KeyPool) GetNextKey() (string, int) {
	kp.mu.Lock()
	defer kp.mu.Unlock()

	// Tìm key healthy và không đang bị overload
	candidates := []int{}
	for i, healthy := range kp.HealthStatus {
		if healthy && kp.KeysInUse[kp.Keys[i]] < 5 { // max 5 concurrent per key
			candidates = append(candidates, i)
		}
	}

	if len(candidates) == 0 {
		// No healthy keys - rotate immediately
		kp.rotateKeys()
		return kp.Keys[0], 0
	}

	// Simple round-robin for now (simpler than weighted selection)
	selectedIdx := candidates[kp.CurrentIndex%len(candidates)]
	kp.CurrentIndex++

	// Track usage
	key := kp.Keys[selectedIdx]
	kp.KeysInUse[key]++
	kp.TotalUsage[selectedIdx]++
	kp.LastUsed[selectedIdx] = time.Now()

	return key, selectedIdx
}

// ReleaseKey giảm count in-use khi request hoàn thành
func (kp *KeyPool) ReleaseKey(key string) {
	kp.mu.Lock()
	defer kp.mu.Unlock()

	if count, exists := kp.KeysInUse[key]; exists {
		if count > 1 {
			kp.KeysInUse[key] = count - 1
		} else {
			delete(kp.KeysInUse, key)
		}
	}
}

// UpdateHealth cập nhật health status của một key
func (kp *KeyPool) UpdateHealth(keyIndex int, success bool, latency time.Duration) {
	kp.mu.Lock()
	defer kp.mu.Unlock()

	if success {
		kp.HealthStatus[keyIndex] = true
		kp.TotalFailures[keyIndex] = 0 // Reset failure count on success
	} else {
		// Check failover threshold
		if !kp.HealthStatus[keyIndex] {
			kp.TotalFailures[keyIndex]++
			if kp.TotalFailures[keyIndex] >= kp.FailoverThreshold {
				kp.rotateKeys()
			}
		} else {
			kp.HealthStatus[keyIndex] = false
			kp.TotalFailures[keyIndex] = 1
		}
	}

	// Update latency metrics if needed
}

// rotateKeys thực hiện rotation key tự động
func (kp *KeyPool) rotateKeys() {
	kp.mu.Lock()
	defer kp.mu.Unlock()

	// Generate new keys (demonstration - trong thực tế sẽ load từ secure storage)
	newKeys := generateNewKeys(kp.MaxKeys)

	// Calculate Weights based on performance
	totalHealthy := 0
	for _, healthy := range kp.HealthStatus {
		if healthy {
			totalHealthy++
		}
	}

	if totalHealthy > 0 {
		weightPerKey := 1.0 / float64(totalHealthy)
		for i := range kp.Weights {
			if kp.HealthStatus[i] {
				kp.Weights[i] = weightPerKey
			} else {
				kp.Weights[i] = 0.0
			}
		}
	}

	kp.Keys = newKeys
	kp.HealthStatus = make([]bool, len(newKeys))
	kp.TotalUsage = make([]int64, len(newKeys))
	kp.KeysInUse = make(map[string]int)
	kp.CurrentIndex = 0

	log.Printf("🔄 [KeyPool] Rotated %d keys", len(newKeys))
}

// startRotationScheduler chạy background rotation
func (kp *KeyPool) startRotationScheduler() {
	ticker := time.NewTicker(kp.RotationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			kp.rotateKeys()
		}
	}
}

// GetStats trả về thống kê performance của keys
func (kp *KeyPool) GetStats() []KeyStats {
	kp.mu.RLock()
	defer kp.mu.RUnlock()

	stats := make([]KeyStats, len(kp.Keys))
	for i, key := range kp.Keys {
		stats[i] = KeyStats{
			Key:        key,
			Weight:     kp.Weights[i],
			SuccessRate: calculateSuccessRate(i),
			UsageCount:  kp.TotalUsage[i],
			LastUsed:    kp.LastUsed[i],
		}
	}

	return stats
}

// Helper functions
func loadKeysFromEnv() []string {
	// TODO: Load từ environment hoặc secure storage
	return []string{
		"AIzaSyB3y5xJ7X8y9K9f2w5X8y9K9f2w5X8y9K9", // Placeholder
	}
}

func generateNewKeys(count int) []string {
	// TODO: Generate từ secure storage
	keys := make([]string, count)
	for i := range keys {
		keys[i] = fmt.Sprintf("AIzaSyB3y5xJ7X8y9K9f2w5X8y9K9f2w%d", i)
	}
	return keys
}

func calculateSuccessRate(keyIndex int) float64 {
	// TODO: Calculate dựa trên success/failure history
	return 1.0 // Placeholder
}

// CreateGeminiClient tạo client với key từ pool
func (kp *KeyPool) CreateGeminiClient() (*genai.Client, string, int, error) {
	key, keyIndex := kp.GetNextKey()

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(key))
	if err != nil {
		kp.UpdateHealth(keyIndex, false, 0)
		return nil, "", 0, err
	}

	return client, key, keyIndex, nil
}