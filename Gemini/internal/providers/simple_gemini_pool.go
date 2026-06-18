package providers

import (
	"context"
	"fmt"
	"log"
	"time"

	"gemini-cli/internal/utils"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// SimpleGeminiProvider là Gemini provider với key pool đơn giản
type SimpleGeminiProvider struct {
	keyPool       *utils.SimpleKeyPool
	model         string
	lastError     error
	errorCount    int
	maxRetries    int
	retryDelay    time.Duration
}

// NewSimpleGeminiProvider tạo Gemini provider với key pool
func NewSimpleGeminiProvider(apiKeys, model string, maxRetries int) (*SimpleGeminiProvider, error) {
	keyPool, err := utils.NewSimpleKeyPool(apiKeys)
	if err != nil {
		return nil, fmt.Errorf("failed to create key pool: %w", err)
	}

	return &SimpleGeminiProvider{
		keyPool:    keyPool,
		model:      model,
		maxRetries: maxRetries,
		retryDelay: 100 * time.Millisecond,
	}, nil
}

// SendMessage gửi message với retry logic
func (gp *SimpleGeminiProvider) SendMessage(ctx context.Context, message string) (string, error) {
	// Try with different keys
	keys := gp.keyPool.GetAllKeys()

	for _, key := range keys {
		// Try each key multiple times
		for attempt := 0; attempt < gp.maxRetries; attempt++ {
			if attempt > 0 {
				time.Sleep(gp.retryDelay * time.Duration(attempt))
			}

			resp, err := gp.generateContentWithKey(ctx, key, message)
			if err == nil {
				gp.errorCount = 0
				return resp, nil
			}

			gp.errorCount++
			log.Printf("Key %s, attempt %d failed: %v", key[:utils.MinValue(10, len(key))]+"...", attempt+1, err)

			// Nếu lỗi là về key, thử key khác
			if utils.IsAPIKeyError(err) {
				continue
			}

			// Other errors, retry with same key
			if attempt == gp.maxRetries-1 {
				return "", fmt.Errorf("after %d attempts: %w", gp.maxRetries, err)
			}
		}
	}

	return "", fmt.Errorf("all keys failed after retries")
}

// generateContentWithKey tạo content với specific key
func (gp *SimpleGeminiProvider) generateContentWithKey(ctx context.Context, key, message string) (string, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(key))
	if err != nil {
		return "", fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	model := client.GenerativeModel(gp.model)
	resp, err := model.GenerateContent(ctx, genai.Text(message))
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	return utils.FormatGeminiResponse(resp), nil
}

// GetRandomKey dùng ngẫu nhiên key
func (gp *SimpleGeminiProvider) GetRandomKey() string {
	return gp.keyPool.GetRandomKey()
}

// GetStats trả về statistics
func (gp *SimpleGeminiProvider) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"keys_count":   gp.keyPool.GetKeysCount(),
		"usage_stats": gp.keyPool.Stats(),
	}
}