package application

import (
	"context"

	"gemini-cli/internal/domain/entities"
	"gemini-cli/internal/domain/interfaces"
)

// AgentService implements the AgentService interface
type AgentService struct {
	orchestrator interfaces.Orchestrator
}

// NewAgentService creates a new agent service
func NewAgentService(orchestrator interfaces.Orchestrator) *AgentService {
	return &AgentService{
		orchestrator: orchestrator,
	}
}

// Process processes a user message and returns AI response
func (a *AgentService) Process(ctx context.Context, req *interfaces.ProcessRequest) (*interfaces.ProcessResponse, error) {
	// Execute orchestrator
	orchResp, err := a.orchestrator.ExecuteReAct(ctx, &interfaces.OrchestrationRequest{
		Message:      req.Message,
		Attachments:  req.Attachments,
		MaxIterations: 20,
		Agent:        req.ForceAgent,
	})

	if err != nil {
		return nil, err
	}

	return &interfaces.ProcessResponse{
		Reply:      orchResp.Reply,
		History:    []entities.Message{}, // Would be populated from orchestrator
		UsedAgent:  orchResp.Agent,
		TokenUsage: orchResp.TokenUsage,
		ExecTime:   orchResp.ExecTime,
		ToolCalls:  orchResp.ToolCalls,
	}, nil
}

// ProcessMessageLegacy maintains backward compatibility
func (a *AgentService) ProcessMessage(ctx context.Context, userInput string, atts []entities.Attachment) (string, error) {
	resp, err := a.Process(ctx, &interfaces.ProcessRequest{
		Message:     userInput,
		Attachments: atts,
	})
	if err != nil {
		return "", err
	}
	return resp.Reply, nil
}

// ProcessMessageStreamLegacy maintains backward compatibility
func (a *AgentService) ProcessMessageStream(ctx context.Context, userInput string, atts []entities.Attachment, onChunk func(string, bool)) error {
	// Simplified streaming
	resp, err := a.ProcessMessage(ctx, userInput, atts)
	if err != nil {
		return err
	}

	// Simulate streaming
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
func (a *AgentService) GetAgent() *entities.Agent {
	// Placeholder
	return entities.NewAgent("default", "Default Agent", "General purpose AI assistant", []string{"general"})
}

// GetHistory returns conversation history
func (a *AgentService) GetHistory() []entities.Message {
	// Placeholder
	return []entities.Message{}
}

// LoadHistory loads conversation history
func (a *AgentService) LoadHistory(msgs []entities.Message) {
	// Placeholder
}

// Reset clears conversation history
func (a *AgentService) Reset() {
	// Placeholder
}

// GetCapabilities returns agent capabilities
func (a *AgentService) GetCapabilities() []string {
	// Placeholder
	return []string{"general"}
}
