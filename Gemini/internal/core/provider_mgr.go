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
// Priority: Gemini primary → OpenRouter fallbacks. Can be overridden via USE_OPENROUTER_ONLY.
func NewProviderManager() *ProviderManager {
	geminiProviders := newGeminiProviders()
	orProviders := newOpenRouterProviders(openRouterKeysFromEnv())

	useOnlyOR := os.Getenv("USE_OPENROUTER_ONLY") == "1" || len(geminiProviders) == 0

	var p providers.Provider
	if useOnlyOR && len(orProviders) > 0 {
		fmt.Println("🚀 [Config] Sử dụng OpenRouter làm primary (bypass Gemini để tránh quota)")
		p = providers.NewMultiProvider(orProviders[0], orProviders[1:])
	} else {
		allProviders := append(geminiProviders, orProviders...)
		if len(allProviders) == 0 {
			allProviders = append(allProviders, newGeminiProvider(""))
		}
		p = providers.NewMultiProvider(allProviders[0], allProviders[1:])
	}

	return &ProviderManager{provider: p}
}

// GetProvider returns the current provider chain (may be MultiProvider wrapping primary + fallbacks).
func (pm *ProviderManager) GetProvider() providers.Provider {
	return pm.provider
}

// SetOpenRouterKeys replaces the provider chain with new OpenRouter keys at runtime.
func (pm *ProviderManager) SetOpenRouterKeys(keys []string) {
	var cleanKeys []string
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		cleanKeys = append(cleanKeys, key)
	}

	orProviders := newOpenRouterProviders(cleanKeys)
	allProviders := append(newGeminiProviders(), orProviders...)
	if len(allProviders) == 0 {
		allProviders = append(allProviders, newGeminiProvider(""))
	}
	pm.provider = providers.NewMultiProvider(allProviders[0], allProviders[1:])

	fmt.Printf("🔑 [Config] Updated OpenRouter keys. Count: %d\n", len(orProviders))
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
func openRouterKeysFromEnv() []string {
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
func newOpenRouterProviders(keys []string) []providers.Provider {
	model := os.Getenv("OPENROUTER_MODEL")
	if model == "" {
		model = "meta-llama/llama-3.3-70b-instruct:free"
	}

	openRouters := make([]providers.Provider, 0, len(keys))
	for _, key := range keys {
		openRouters = append(openRouters, &providers.OpenRouterProvider{
			APIKey: key,
			Model:  model,
		})
	}
	return openRouters
}

// normalizeGeminiModel ensures the model name starts with "models/" prefix.
func normalizeGeminiModel(model string) string {
	normalized := strings.TrimSpace(model)
	if normalized == "" {
		return "gemini-flash-latest"
	}
	if !strings.HasPrefix(normalized, "models/") {
		return "models/" + normalized
	}
	return normalized
}
