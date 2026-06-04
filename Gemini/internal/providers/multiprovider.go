package providers

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"gemini-cli/internal/api"
	"gemini-cli/internal/models/messaging"
)

type MultiProvider struct {
	mu               sync.Mutex
	primary          Provider
	fallbacks        []Provider
	currentIdx       int
	primaryFailures  int     // số lần primary fail liên tiếp
	skipPrimaryUntil int     // skip primary cho N lần gọi tới (để tránh quota)
}

func NewMultiProvider(primary Provider, fallbacks []Provider) *MultiProvider {
	return &MultiProvider{
		primary:   primary,
		fallbacks: fallbacks,
	}
}

func (m *MultiProvider) GenerateText(systemPrompt, userPrompt string) (string, error) {
	m.mu.Lock()
	skipPrimary := m.skipPrimaryUntil > 0
	m.mu.Unlock()

	if skipPrimary && len(m.fallbacks) > 0 {
		raw, err := m.tryFallbacksOnlyText(systemPrompt, userPrompt)
		if err == nil {
			return raw, nil
		}
		api.BroadcastLog("Fallback failed. Trying primary anyway (GenerateText)...", "routing")
		fmt.Println("🔄 [Fallback] Fallback failed. Trying primary anyway (GenerateText)...")
	}

	raw, err := m.primary.GenerateText(systemPrompt, userPrompt)
	if err == nil {
		m.mu.Lock()
		m.primaryFailures = 0
		m.skipPrimaryUntil = 0
		m.mu.Unlock()
		return raw, nil
	}

	isQuota := isQuotaOrRateLimitError(err)
	fmt.Printf("⚠️ [Fallback] Primary provider error (GenerateText): %v\n", err)

	if isQuota {
		m.mu.Lock()
		m.primaryFailures++
		m.skipPrimaryUntil = 4 + (m.primaryFailures / 2)
		if m.skipPrimaryUntil > 10 {
			m.skipPrimaryUntil = 10
		}
		m.mu.Unlock()
	}

	return m.tryFallbacksOnlyText(systemPrompt, userPrompt)
}

func (m *MultiProvider) tryFallbacksOnlyText(systemPrompt, userPrompt string) (string, error) {
	numFallbacks := len(m.fallbacks)
	if numFallbacks == 0 {
		return "", fmt.Errorf("no fallbacks configured")
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

		var raw string
		raw, lastErr = p.GenerateText(systemPrompt, userPrompt)
		if lastErr == nil {
			m.mu.Lock()
			if m.skipPrimaryUntil > 0 {
				m.skipPrimaryUntil /= 2
			}
			m.mu.Unlock()
			return raw, nil
		}
		api.BroadcastLog(fmt.Sprintf("Fallback #%d error: %v", activeIdx+1, lastErr), "error")
		fmt.Printf("⚠️ [Fallback] Fallback #%d error: %v\n", activeIdx+1, lastErr)
	}

	return "", fmt.Errorf("Tất cả các dịch vụ (Primary & %d Fallbacks) đều gặp lỗi hoặc hết hạn mức (Quota). Lỗi cuối cùng: %v", numFallbacks, lastErr)
}

func (m *MultiProvider) Generate(ctx context.Context, req messaging.Request) (messaging.Message, error) {
	m.mu.Lock()
	skipPrimary := m.skipPrimaryUntil > 0
	if skipPrimary {
		m.skipPrimaryUntil--
	}
	m.mu.Unlock()

	// Nếu đang skip primary do quota gần đây → dùng fallback ngay
	if skipPrimary && len(m.fallbacks) > 0 {
		api.BroadcastLog("Primary đang bị rate limit gần đây, ưu tiên fallback...", "routing")
		aiMessage, err := m.tryFallbacksOnly(ctx, req)
		if err == nil {
			return aiMessage, nil
		}
		api.BroadcastLog("Fallback failed. Trying primary anyway...", "routing")
		fmt.Println("🔄 [Fallback] Fallback failed. Trying primary anyway...")
	}

	// Try primary first
	api.BroadcastLog("Calling primary provider...", "process")
	aiMessage, err := m.primary.Generate(ctx, req)
	if err == nil {
		// Thành công → reset failure counter
		m.mu.Lock()
		m.primaryFailures = 0
		m.skipPrimaryUntil = 0
		m.mu.Unlock()
		return aiMessage, nil
	}

	isQuotaError := isQuotaOrRateLimitError(err)

	api.BroadcastLog(fmt.Sprintf("Primary error: %v", err), "error")
	fmt.Printf("⚠️ [Fallback] Primary error (Generate): %v\n", err)

	if isQuotaError {
		// Tăng skip để tránh spam primary
		m.mu.Lock()
		m.primaryFailures++
		// Skip primary cho 4-6 lượt tiếp theo (giảm tải quota)
		m.skipPrimaryUntil = 5 + (m.primaryFailures / 2)
		if m.skipPrimaryUntil > 12 {
			m.skipPrimaryUntil = 12
		}
		m.mu.Unlock()
		fmt.Printf("⏳ [RateLimit] Phát hiện quota Gemini. Sẽ ưu tiên fallback cho %d lượt tiếp theo.\n", 5+(m.primaryFailures/2))
	}

	// Fallback logic
	return m.tryFallbacksOnly(ctx, req)
}

func (m *MultiProvider) tryFallbacksOnly(ctx context.Context, req messaging.Request) (messaging.Message, error) {
	numFallbacks := len(m.fallbacks)
	if numFallbacks == 0 {
		return messaging.Message{}, fmt.Errorf("Dịch vụ chính gặp lỗi và không có dịch vụ dự phòng")
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
			// Fallback thành công → giảm dần skip primary
			m.mu.Lock()
			if m.skipPrimaryUntil > 0 {
				m.skipPrimaryUntil = m.skipPrimaryUntil / 2
			}
			m.mu.Unlock()
			return aiMessage, nil
		}

		lastErr = err
		api.BroadcastLog(fmt.Sprintf("Fallback #%d error: %v", activeIdx+1, err), "error")
		fmt.Printf("⚠️ [Fallback] Fallback #%d error: %v\n", activeIdx+1, err)
	}

	return messaging.Message{}, fmt.Errorf("Tất cả các dịch vụ (%d dự phòng) đều thất bại. Có thể do hết hạn mức (Quota) hoặc sự cố kết nối. Lỗi cuối cùng: %v", numFallbacks, lastErr)
}

func isQuotaOrRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "quota") ||
		strings.Contains(msg, "rate") ||
		strings.Contains(msg, "429") ||
		strings.Contains(msg, "exceeded")
}
