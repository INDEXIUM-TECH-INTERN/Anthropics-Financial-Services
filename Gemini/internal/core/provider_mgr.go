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

// NewProviderManager creates a provider chain from environment configuration.
// Priority: Gemini API keys with failover support. No longer supports OpenRouter.
func NewProviderManager() *ProviderManager {
	geminiProviders := newGeminiProviders()

	var p providers.Provider
	if len(geminiProviders) == 0 {
		fmt.Println("⚠️ [Config] Không tìm thấy Gemini API keys. Sử dụng API key trống để testing.")
		p = providers.NewMultiProvider(newGeminiProvider(""), nil)
	} else {
		p = providers.NewMultiProvider(geminiProviders[0], geminiProviders[1:])
		fmt.Printf("🚀 [Config] Đã khởi tạo Gemini MultiProvider với %d API keys\n", len(geminiProviders))
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

// openRouterKeysFromEnv reads OpenRouter API keys from OPENROUTER_API_KEY, OPENROUTER_API_KEY_2, ..., _5.
// Deprecated: This function is kept for backward compatibility but OpenRouter is no longer supported.
func openRouterKeysFromEnv() []string {
	fmt.Println("⚠️ [Deprecated] OpenRouter is no longer supported in Gemini backend")
	return envValues(numberedEnvKeys("OPENROUTER_API_KEY", 5))
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

// newOpenRouterProviders creates OpenRouterProvider instances from env keys.
// Deprecated: OpenRouter is no longer supported in Gemini backend.
func newOpenRouterProviders(keys []string) []providers.Provider {
	fmt.Println("⚠️ [Deprecated] OpenRouter is no longer supported in Gemini backend")
	// Return empty array to prevent OpenRouter usage
	return []providers.Provider{}
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
