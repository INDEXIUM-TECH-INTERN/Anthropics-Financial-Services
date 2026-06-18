package entities

import (
	"context"
	"fmt"
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