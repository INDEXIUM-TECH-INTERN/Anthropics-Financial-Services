package providers

import (
	"context"
	"fmt"
	"gemini-cli/internal/api"
	"gemini-cli/internal/models/messaging"
	"sync"
)

type MultiProvider struct {
	mu         sync.Mutex
	primary    Provider
	fallbacks  []Provider
	currentIdx int
}

func NewMultiProvider(primary Provider, fallbacks []Provider) *MultiProvider {
	return &MultiProvider{
		primary:   primary,
		fallbacks: fallbacks,
	}
}

func (m *MultiProvider) GenerateText(systemPrompt, userPrompt string) (string, error) {
	// Try primary first
	raw, err := m.primary.GenerateText(systemPrompt, userPrompt)
	if err == nil {
		return raw, nil
	}
	fmt.Printf("⚠️ [Fallback] Primary provider error (GenerateText): %v\n", err)

	// Fallback logic
	numFallbacks := len(m.fallbacks)
	if numFallbacks == 0 {
		return "", fmt.Errorf("primary error and no fallbacks configured: %w", err)
	}

	var lastErr error
	for i := 0; i < numFallbacks; i++ {
		m.mu.Lock()
		activeIdx := m.currentIdx
		m.currentIdx = (m.currentIdx + 1) % numFallbacks
		m.mu.Unlock()

		p := m.fallbacks[activeIdx]
		api.BroadcastLog(fmt.Sprintf("Primary error. Trying fallback #%d (GenerateText)...", activeIdx+1), "routing")
		fmt.Printf("🔄 [Fallback] Trying fallback #%d (GenerateText)...\n", activeIdx+1)

		raw, lastErr = p.GenerateText(systemPrompt, userPrompt)
		if lastErr == nil {
			return raw, nil
		}
		api.BroadcastLog(fmt.Sprintf("Fallback #%d error: %v", activeIdx+1, lastErr), "error")
		fmt.Printf("⚠️ [Fallback] Fallback #%d error: %v\n", activeIdx+1, lastErr)
	}

	return "", fmt.Errorf("all %d fallbacks failed: %v", numFallbacks, lastErr)
}

func (m *MultiProvider) Generate(ctx context.Context, req messaging.Request) (messaging.Message, error) {
	// Try primary first
	api.BroadcastLog("Calling primary provider...", "process")
	aiMessage, err := m.primary.Generate(ctx, req)
	if err == nil {
		return aiMessage, nil
	}

	api.BroadcastLog(fmt.Sprintf("Primary error: %v", err), "error")
	fmt.Printf("⚠️ [Fallback] Primary error (Generate): %v\n", err)

	// Fallback logic
	numFallbacks := len(m.fallbacks)
	if numFallbacks == 0 {
		return messaging.Message{}, fmt.Errorf("primary error and no fallbacks: %w", err)
	}

	var lastErr error
	for i := 0; i < numFallbacks; i++ {
		m.mu.Lock()
		activeIdx := m.currentIdx
		m.currentIdx = (m.currentIdx + 1) % numFallbacks
		m.mu.Unlock()

		p := m.fallbacks[activeIdx]
		api.BroadcastLog(fmt.Sprintf("Using fallback provider #%d...", activeIdx+1), "process")
		fmt.Printf("🔄 [Fallback] Trying fallback #%d (Generate)...\n", activeIdx+1)

		aiMessage, err := p.Generate(ctx, req)
		if err == nil {
			return aiMessage, nil
		}

		lastErr = err
		api.BroadcastLog(fmt.Sprintf("Fallback #%d error: %v", activeIdx+1, err), "error")
		fmt.Printf("⚠️ [Fallback] Fallback #%d error: %v\n", activeIdx+1, err)
	}

	return messaging.Message{}, fmt.Errorf("all %d fallbacks failed: %v", numFallbacks, lastErr)
}
