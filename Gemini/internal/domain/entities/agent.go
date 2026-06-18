package entities

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"gemini-cli/internal/models/messaging"
)

// Agent represents a specialized AI agent with specific capabilities
type Agent struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Capabilities []string          `json:"capabilities"`
	Config       map[string]any    `json:"config"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	Mu           sync.RWMutex      `json:"-"`         // Mutex for thread safety
	UserInput    string            `json:"-"`         // Current user input
	Conversation *ContextWindow    `json:"-"`         // Conversation history
	SystemPrompt string            `json:"-"`         // System prompt
	HandoffPlan  string            `json:"-"`         // Handoff plan for routing
	Provider     LLMProvider       `json:"-"`         // LLM provider for routing
}

// NewAgent creates a new agent instance
func NewAgent(id, name, description string, capabilities []string) *Agent {
	now := time.Now()
	return &Agent{
		ID:           id,
		Name:         name,
		Description:  description,
		Capabilities: capabilities,
		Config:       make(map[string]any),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// UpdateConfig updates agent configuration
func (a *Agent) UpdateConfig(config map[string]any) {
	a.Mu.Lock()
	defer a.Mu.Unlock()

	a.Config = config
	a.UpdatedAt = time.Now()
}

// AddCapability adds a new capability to the agent
func (a *Agent) AddCapability(capability string) {
	a.Mu.Lock()
	defer a.Mu.Unlock()

	for _, cap := range a.Capabilities {
		if cap == capability {
			return
		}
	}
	a.Capabilities = append(a.Capabilities, capability)
	a.UpdatedAt = time.Now()
}

// HasCapability checks if agent has a specific capability
func (a *Agent) HasCapability(capability string) bool {
	a.Mu.RLock()
	defer a.Mu.RUnlock()

	for _, cap := range a.Capabilities {
		if cap == capability {
			return true
		}
	}
	return false
}

// GetID returns agent ID
func (a *Agent) GetID() string {
	return a.ID
}

// GetName returns agent name
func (a *Agent) GetName() string {
	return a.Name
}

// GetDescription returns agent description
func (a *Agent) GetDescription() string {
	return a.Description
}

// GetCapabilities returns agent capabilities
func (a *Agent) GetCapabilities() []string {
	a.Mu.RLock()
	defer a.Mu.RUnlock()

	cp := make([]string, len(a.Capabilities))
	copy(cp, a.Capabilities)
	return cp
}

// GetConfig returns agent config
func (a *Agent) GetConfig() map[string]any {
	a.Mu.RLock()
	defer a.Mu.RUnlock()

	cp := make(map[string]any, len(a.Config))
	for k, v := range a.Config {
		cp[k] = v
	}
	return cp
}

// AppendUserTextInternal appends user text to the agent's internal state
func (a *Agent) AppendUserTextInternal(text string, attachments []Attachment) {
	a.Mu.Lock()
	defer a.Mu.Unlock()

	a.UserInput = text

	// Add user message to conversation history
	if a.Conversation == nil {
		a.Conversation = NewContextWindow(100, 10000, 50)
	}

	// Add message to context window
	history := a.Conversation.GetHistory()
	if len(history) >= a.Conversation.MaxMessages {
		// Remove oldest message
		a.Conversation.History = a.Conversation.History[1:]
	}

	a.Conversation.History = append(a.Conversation.History, Message{
		ID:           generateID(),
		Role:         "user",
		Content:      text,
		Timestamp:    time.Now(),
		Attachments:  attachments,
	})
	a.UpdatedAt = time.Now()
}

// GetConversation returns the conversation associated with the agent
func (a *Agent) GetConversation() *ContextWindow {
	a.Mu.RLock()
	defer a.Mu.RUnlock()

	if a.Conversation == nil {
		a.Conversation = NewContextWindow(100, 10000, 50)
	}

	// Return a copy to prevent external modification
	cp := &ContextWindow{
		History:     make([]Message, len(a.Conversation.History)),
		MaxMessages: a.Conversation.MaxMessages,
		MaxTokens:   a.Conversation.MaxTokens,
		WindowSize:  a.Conversation.WindowSize,
	}
	copy(cp.History, a.Conversation.History)
	return cp
}

// GetUserInput returns the current user input
func (a *Agent) GetUserInput() string {
	a.Mu.RLock()
	defer a.Mu.RUnlock()
	return a.UserInput
}

// SetUserInput sets the current user input
func (a *Agent) SetUserInput(input string) {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	a.UserInput = input
}

// Lock locks the agent's mutex
func (a *Agent) Lock() {
	a.Mu.Lock()
}

// Unlock unlocks the agent's mutex
func (a *Agent) Unlock() {
	a.Mu.Unlock()
}

// GetSystemPrompt returns the system prompt
func (a *Agent) GetSystemPrompt() string {
	a.Mu.RLock()
	defer a.Mu.RUnlock()
	return a.SystemPrompt
}

// SetSystemPrompt sets the system prompt
func (a *Agent) SetSystemPrompt(prompt string) {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	a.SystemPrompt = prompt
}

// GetHandoffPlan returns the handoff plan
func (a *Agent) GetHandoffPlan() string {
	a.Mu.RLock()
	defer a.Mu.RUnlock()
	return a.HandoffPlan
}

// SetHandoffPlan sets the handoff plan
func (a *Agent) SetHandoffPlan(plan string) {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	a.HandoffPlan = plan
}

// generateID generates a unique ID
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// MockProvider is a mock implementation of LLMProvider for testing
type MockProvider struct{}

func (m *MockProvider) Generate(context.Context, *LLMRequest) (*LLMResponse, error) {
	return &LLMResponse{
		Message: Message{
			Role:    string(messaging.RoleAssistant),
			Content: "Mock response",
		},
		Model: "mock-model",
	}, nil
}

func (m *MockProvider) GenerateStream(context.Context, *LLMRequest, func(StreamChunk)) error {
	return nil
}

func (m *MockProvider) GetModelInfo() ModelInfo {
	return ModelInfo{
		ID:           "mock-model",
		Name:         "Mock Model",
		Description:  "Mock model for testing",
		Capabilities: []string{"chat"},
		MaxTokens:    4000,
	}
}

func (m *MockProvider) SetModel(string) error {
	return nil
}

func (m *MockProvider) GetAvailableModels() []ModelInfo {
	return []ModelInfo{
		{
			ID:           "mock-model",
			Name:         "Mock Model",
			Description:  "Mock model for testing",
			Capabilities: []string{"chat"},
			MaxTokens:    4000,
		},
	}
}

func (m *MockProvider) IsHealthy() bool {
	return true
}

// CheckTimeKnowledgeIssues kiểm tra và cảnh báo các vấn đề về thời gian
func (a *Agent) CheckTimeKnowledgeIssues(text string) string {
	now := time.Now()
	currentYear := now.Year()
	warnings := []string{}
	warningSet := make(map[string]bool) // Trùng lặp warnings

	// 1. Kiểm tra các năm cũ (2024 và trước đó khi đã có năm mới)
	if currentYear > 2024 {
		yearMatches := regexp.MustCompile(`\b2024\b`).FindAllString(text, -1)
		if len(yearMatches) > 0 {
			warningSet["- Dữ liệu năm 2024 có thể không còn mới nhất"] = true
		}

		// Kiểm tra các năm cũ khác
		oldYears := []int{2023, 2022, 2021}
		for _, year := range oldYears {
			yearMatches := regexp.MustCompile(`\b`+strconv.Itoa(year)+`\b`).FindAllString(text, -1)
			if len(yearMatches) > 0 {
				warningSet[fmt.Sprintf("- Tham chiếu đến năm %d có thể không cập nhật", year)] = true
			}
		}
	}

	// 2. Kiểm tra các biểu thức thời gian không hợp lý
	illogicalPatterns := []string{
		`3 năm trước`, `2 năm trước`, `1 năm trước`,
		`6 tháng trước`, `3 tháng trước`, `1 tháng trước`,
	}
	for _, pattern := range illogicalPatterns {
		if strings.Contains(text, pattern) {
			// Kiểm tra xem kết quả có phải là năm cũ không
			for _, year := range []int{2024, 2023, 2022} {
				if strings.Contains(text, strconv.Itoa(year)) {
					warningSet[fmt.Sprintf("- Biểu thức '%s' đề cập đến năm %d có thể không chính xác", pattern, year)] = true
					break
				}
			}
		}
	}

	// 3. Kiểm tra "gần đây" với các năm cũ
	recentPatterns := []string{"gần nhất", "gần đây", "hiện tại"}
	for _, pattern := range recentPatterns {
		if strings.Contains(text, pattern) {
			// Nếu chứa các pattern này nhưng lại đề cập đến năm 2024
			if strings.Contains(text, "2024") && currentYear > 2024 {
				warningSet[fmt.Sprintf("- Từ '%s' cần được xác nhận lại với dữ liệu mới nhất", pattern)] = true
			}
		}
	}

	// 4. Kiểm tra giờ giao dịch
	if strings.Contains(text, "giờ giao dịch") {
		hour := now.Hour()
		weekday := now.Weekday()
		if !(weekday >= time.Monday && weekday <= time.Friday && hour >= 9 && hour < 15) {
			warningSet["- Ngoài giờ giao dịch chính (9:00-15:00 từ thứ 2 đến thứ 6)"] = true
		}
	}

	// 5. Sử dụng time parser để validate các biểu thức thời gian cụ thể
	timeExpressions := regexp.MustCompile(`\d{1,2}[/-]\d{1,2}[/-]\d{4}|\d{4}`).FindAllString(text, -1)
	for _, expr := range timeExpressions {
		// Nếu là năm và là năm cũ
		if len(expr) == 4 {
			year, err := strconv.Atoi(expr)
			if err == nil && year < currentYear-1 {
				warningSet[fmt.Sprintf("- Dữ liệu năm %d có thể không còn mới", year)] = true
			}
		}
	}

	// Chuyển map thành slice để tránh trùng lặp
	for warning := range warningSet {
		warnings = append(warnings, warning)
	}

	if len(warnings) > 0 {
		result := "*⚠️ CẢNH BÁO VỀ THỜI GIAN:*\n"
		result += strings.Join(warnings, "\n")
		result += fmt.Sprintf("\n\nHiện tại là: %s (%d)\n", now.Format("02/01/2006"), currentYear)
		result += "Vui lòng kiểm tra với nguồn dữ liệu mới nhất.\n\n"
		return result
	}

	return ""
}