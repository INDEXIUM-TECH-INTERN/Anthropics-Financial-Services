package core

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"gemini-cli/internal/providers"
)

// ProviderManager handles LLM provider initialization, key rotation, and failover.
// It is owned by Agent and accessed under Agent's RWMutex.
type ProviderManager struct {
	mu       sync.RWMutex
	provider providers.Provider
	keys     []string
	current  int // current key index for round-robin
}

// NewProviderManager creates a provider chain from environment configuration.
// Priority: Gemini API keys with round-robin rotation. No fallback providers.
func NewProviderManager() *ProviderManager {
	keys := newGeminiKeys()

	var p providers.Provider
	if len(keys) == 0 {
		fmt.Println("⚠️ [Config] Không tìm thấy Gemini API keys. Sử dụng API key trống để testing.")
		p = providers.NewGeminiProvider("")
	} else {
		// For multiple keys, create a simple provider that rotates keys
		if len(keys) == 1 {
			p = providers.NewGeminiProvider(keys[0])
		} else {
			// Use Gemini provider with key pool support
			p = providers.NewGeminiProviderWithPool(keys)
		}
		fmt.Printf("🚀 [Config] Đã khởi tạo Gemini Provider với %d API keys (round-robin)\n", len(keys))
	}

	return &ProviderManager{
		provider: p,
		keys:     keys,
	}
}

// GetProvider returns the current provider.
func (pm *ProviderManager) GetProvider() providers.Provider {
	return pm.provider
}

// SetGeminiKeys replaces the current key pool with new keys.
func (pm *ProviderManager) SetGeminiKeys(keys []string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var cleanKeys []string
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key != "" {
			cleanKeys = append(cleanKeys, key)
		}
	}

	pm.keys = cleanKeys
	pm.current = 0

	if len(cleanKeys) > 0 {
		fmt.Printf("🔑 [Config] Updated %d Gemini keys (round-robin rotation)\n", len(cleanKeys))
	}
}

// GetNextKey returns the next key in round-robin fashion.
func (pm *ProviderManager) GetNextKey() string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if len(pm.keys) == 0 {
		return ""
	}

	key := pm.keys[pm.current]
	pm.current = (pm.current + 1) % len(pm.keys)
	return key
}

// numberedEnvKeys generates a list of env var names: base, base_2, base_3, ..., base_max.
func numberedEnvKeys(base string, max int) []string {
	keys := []string{base}
	for i := 2; i <= max; i++ {
		keys = append(keys, fmt.Sprintf("%s_%d", base, i))
	}
	return keys
}

// envValues reads non-disabled env var values for the given keys.
func envValues(keys []string) []string {
	var values []string
	for _, key := range keys {
		value := strings.TrimSpace(os.Getenv(key))
		if value == "" || value == "disabled" {
			continue
		}
		values = append(values, value)
		fmt.Printf("🔑 [Config] Loaded key from %s\n", key)
	}
	return values
}

// newGeminiKeys reads Gemini API keys from GEMINI_API_KEY env vars.
func newGeminiKeys() []string {
	return envValues(numberedEnvKeys("GEMINI_API_KEY", 5))
}

// normalizeGeminiModel ensures the model name starts with "models/" prefix.
func normalizeGeminiModel(model string) string {
	normalized := strings.TrimSpace(model)
	if normalized == "" {
		return "models/gemini-3.1-flash-lite" // Updated default for 2026
	}
	if !strings.HasPrefix(normalized, "models/") {
		return "models/" + normalized
	}
	return normalized
}
