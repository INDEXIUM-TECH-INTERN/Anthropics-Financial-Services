package core

import (
	"fmt"
	"os"
	"strings"

	"gemini-cli/internal/providers"
)

// ProviderManager handles LLM provider initialization, key rotation, and failover.
// It is owned by Agent and accessed under Agent's RWMutex.
type ProviderManager struct {
	provider providers.Provider
}

// NewProviderManager creates a provider from environment configuration.
// Only supports Gemini API keys with single provider (no fallbacks).
func NewProviderManager() *ProviderManager {
	geminiProviders := newGeminiProviders()

	var p providers.Provider
	if len(geminiProviders) == 0 {
		fmt.Println("⚠️ [Config] Không tìm thấy Gemini API keys. Sử dụng API key trống để testing.")
		p = newGeminiProvider("")
	} else {
		// Cast để truy cập APIKey từ interface
		if gemProvider, ok := geminiProviders[0].(*providers.GeminiProvider); ok {
			p = newGeminiProvider(gemProvider.APIKey)
		} else {
			p = newGeminiProvider("")
		}
		fmt.Printf("🚀 [Config] Đã khởi tạo Gemini Provider với %d API key(s)\n", len(geminiProviders))
	}

	return &ProviderManager{provider: p}
}

// GetProvider returns the current provider chain (may be MultiProvider wrapping primary + fallbacks).
func (pm *ProviderManager) GetProvider() providers.Provider {
	return pm.provider
}

// SetGeminiKeys replaces the provider chain with new Gemini API keys at runtime.
func (pm *ProviderManager) SetGeminiKeys(keys []string) {
	var cleanKeys []string
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		cleanKeys = append(cleanKeys, key)
	}

	if len(cleanKeys) > 0 {
		// TODO: Implement key pool update
		fmt.Printf("🔑 [Config] Updated %d Gemini keys (key pool not implemented yet)\n", len(cleanKeys))
	}
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

// newGeminiProviders creates GeminiProvider instances from GEMINI_API_KEY env vars.
func newGeminiProviders() []providers.Provider {
	keys := envValues(numberedEnvKeys("GEMINI_API_KEY", 5))
	geminis := make([]providers.Provider, 0, len(keys))
	for _, key := range keys {
		geminis = append(geminis, newGeminiProvider(key))
	}
	return geminis
}

// newGeminiProvider creates a single GeminiProvider with normalized model name.
func newGeminiProvider(apiKey string) *providers.GeminiProvider {
	return &providers.GeminiProvider{
		APIKey: apiKey,
		Model:  normalizeGeminiModel(os.Getenv("GEMINI_MODEL")),
	}
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
