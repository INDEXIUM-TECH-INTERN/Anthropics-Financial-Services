package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gemini-cli/internal/domain/entities"
	"gemini-cli/internal/domain/interfaces"
)

// AgentServiceImpl implements the AgentService interface
type AgentServiceImpl struct {
	agent       *entities.Agent
	conversation *entities.ContextWindow
	orchestrator interfaces.Orchestrator
	provider    entities.LLMProvider
	mu          sync.RWMutex
}

// NewAgentService creates a new agent service
func NewAgentService(agent *entities.Agent, conversation *entities.ContextWindow, orchestrator interfaces.Orchestrator, provider entities.LLMProvider) *AgentServiceImpl {
	return &AgentServiceImpl{
		agent:        agent,
		conversation: conversation,
		orchestrator: orchestrator,
		provider:     provider,
	}
}

// Process processes a user message with full domain logic
func (s *AgentServiceImpl) Process(ctx context.Context, req *interfaces.ProcessRequest) (*interfaces.ProcessResponse, error) {
	start := time.Now()

	// Get conversation history
	history := s.conversation.GetHistory()

	// Prepare orchestration request
	orchReq := &interfaces.OrchestrationRequest{
		Message:      req.Message,
		Attachments:  req.Attachments,
		MaxIterations: 20,
		Agent:        req.ForceAgent,
		Context: map[string]any{
			"chat_id":    req.ChatID,
			"user_id":    "anonymous",
			"session_id": "session-" + generateID(),
		},
	}

	// Execute ReAct loop
	orchResp, err := s.orchestrator.ExecuteReAct(ctx, orchReq)
	if err != nil {
		return nil, fmt.Errorf("orchestration failed: %w", err)
	}

	// Add assistant response to conversation
	assistantMsg := entities.Message{
		ID:         generateID(),
		Role:       "assistant",
		Content:    orchResp.Reply,
		ToolCalls:  orchResp.ToolCalls,
		Timestamp:  time.Now(),
	}

	// Add messages to conversation
	s.conversation.History = append(history, assistantMsg)

	// Keep only last MaxMessages messages
	if len(s.conversation.History) > s.conversation.MaxMessages {
		s.conversation.History = s.conversation.History[len(s.conversation.History)-s.conversation.MaxMessages:]
	}

	// Return response
	return &interfaces.ProcessResponse{
		Reply:      orchResp.Reply,
		History:    s.conversation.GetHistory(),
		UsedAgent:  orchResp.Agent,
		TokenUsage: orchResp.TokenUsage,
		ExecTime:   time.Since(start).Milliseconds(),
		ToolCalls:  orchResp.ToolCalls,
	}, nil
}

// ProcessMessageLegacy maintains backward compatibility
func (s *AgentServiceImpl) ProcessMessage(ctx context.Context, userInput string, atts []entities.Attachment) (string, error) {
	resp, err := s.Process(ctx, &interfaces.ProcessRequest{
		Message:     userInput,
		Attachments: atts,
	})
	if err != nil {
		return "", err
	}
	return resp.Reply, nil
}

// ProcessMessageStreamLegacy maintains backward compatibility
func (s *AgentServiceImpl) ProcessMessageStream(ctx context.Context, userInput string, atts []entities.Attachment, onChunk func(string, bool)) error {
	// For now, implement as non-streaming with simulated chunking
	resp, err := s.ProcessMessage(ctx, userInput, atts)
	if err != nil {
		return err
	}

	// Simulate streaming by splitting into chunks
	chunkSize := 3
	runes := []rune(resp)
	for i := 0; i < len(runes); i += chunkSize {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		onChunk(string(runes[i:end]), false)
	}
	onChunk("", true)
	return nil
}

// GetAgent returns the current agent
func (s *AgentServiceImpl) GetAgent() *entities.Agent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.agent
}

// GetHistory returns conversation history
func (s *AgentServiceImpl) GetHistory() []entities.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.conversation.GetHistory()
}

// LoadHistory loads conversation history
func (s *AgentServiceImpl) LoadHistory(msgs []entities.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conversation.History = make([]entities.Message, len(msgs))
	copy(s.conversation.History, msgs)
	s.conversation.UpdatedAt = time.Now()
}

// Reset clears conversation history
func (s *AgentServiceImpl) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conversation.History = []entities.Message{}
	s.conversation.UpdatedAt = time.Now()
}

// GetCapabilities returns agent capabilities
func (s *AgentServiceImpl) GetCapabilities() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.agent.Capabilities
}

// SetAgent sets the current agent
func (s *AgentServiceImpl) SetAgent(agent *entities.Agent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.agent = agent
	s.conversation.UpdatedAt = time.Now()
}

// UpdateSession updates session information
func (s *AgentServiceImpl) UpdateSession(ctx context.Context, updates map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// ContextWindow doesn't have session management - just update updated_at
	s.conversation.UpdatedAt = time.Now()
	return nil
}

// GetConversationStats returns conversation statistics
func (s *AgentServiceImpl) GetConversationStats() ConversationStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := ConversationStats{
		TotalMessages: len(s.conversation.History),
		TotalTokens:   s.conversation.GetTotalTokens(),
		Agent:         s.agent.ID,
		CreatedAt:     time.Now(), // Default since ContextWindow doesn't track created_at
		LastActive:    time.Now(), // Default since ContextWindow doesn't track last_active
	}

	// Count tool calls
	for _, msg := range s.conversation.History {
		stats.ToolCalls = len(msg.ToolCalls)
	}

	return stats
}

// ConversationStats represents conversation statistics
type ConversationStats struct {
	TotalMessages int           `json:"total_messages"`
	TotalTokens   int           `json:"total_tokens"`
	ToolCalls     int           `json:"tool_calls"`
	Agent         string        `json:"agent"`
	CreatedAt     time.Time     `json:"created_at"`
	LastActive    time.Time     `json:"last_active"`
}

// generateID generates a unique ID
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}