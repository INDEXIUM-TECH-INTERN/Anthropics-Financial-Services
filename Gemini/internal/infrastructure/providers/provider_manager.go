package providers

import (
	"fmt"
	"sync"

	"gemini-cli/internal/domain/entities"
)

// ProviderManager manages LLM providers with failover support
type ProviderManager struct {
	providers  []entities.LLMProvider
	currentIdx int
	mu         sync.RWMutex
}

// NewProviderManager creates a new provider manager
func NewProviderManager() *ProviderManager {
	pm := &ProviderManager{}

	// Simplified: Would load from config and create providers
	// For now, return empty provider
	return pm
}

// GetProvider returns the current provider
func (pm *ProviderManager) GetProvider() entities.LLMProvider {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if len(pm.providers) == 0 {
		return nil
	}

	return pm.providers[pm.currentIdx]
}

// GetProviderWithFailover gets a provider and falls back if it fails
func (pm *ProviderManager) GetProviderWithFailover() entities.LLMProvider {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if len(pm.providers) == 0 {
		return nil
	}

	// Try current provider first
	current := pm.providers[pm.currentIdx]
	if current != nil {
		return current
	}

	// Fallback to first provider
	return pm.providers[0]
}

// CycleProvider rotates to next provider
func (pm *ProviderManager) CycleProvider() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if len(pm.providers) <= 1 {
		return
	}

	pm.currentIdx = (pm.currentIdx + 1) % len(pm.providers)
	fmt.Printf("🔄 [ProviderManager] Cycled to provider %d\n", pm.currentIdx+1)
}

// SetProvider sets a specific provider by index
func (pm *ProviderManager) SetProvider(index int) {
	pm.mu.Lock()
	defer pm.mu.RUnlock()

	if index >= 0 && index < len(pm.providers) {
		pm.currentIdx = index
		fmt.Printf("🔄 [ProviderManager] Set to provider %d\n", index+1)
	}
}

// GetAvailableProviders returns all available providers
func (pm *ProviderManager) GetAvailableProviders() []entities.ModelInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var models []entities.ModelInfo
	for _, provider := range pm.providers {
		models = append(models, provider.GetAvailableModels()...)
	}
	return models
}

// GetDefaultProvider returns the configured default provider
func (pm *ProviderManager) GetDefaultProvider() entities.LLMProvider {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for _, provider := range pm.providers {
		// Simplified - would check provider ID vs config.DefaultProvider
		return provider
	}

	return nil
}

// IsHealthy checks if all providers are healthy
func (pm *ProviderManager) IsHealthy() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for _, provider := range pm.providers {
		if !provider.IsHealthy() {
			return false
		}
	}
	return true
}
