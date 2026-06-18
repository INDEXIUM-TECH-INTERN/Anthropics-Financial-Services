package interfaces

import (
	"context"

	"gemini-cli/internal/domain/entities"
)

// Orchestrator defines the interface for orchestration operations
type Orchestrator interface {
	// ExecuteReAct executes the ReAct loop
	ExecuteReAct(ctx context.Context, req *OrchestrationRequest) (*OrchestrationResponse, error)

	// SelectAgent selects the appropriate agent for the task
	SelectAgent(ctx context.Context, input string) (*entities.Agent, error)

	// Handoff performs agent handoff
	Handoff(ctx context.Context, fromAgent, toAgent string, payload interface{}) error

	// GetCapabilities returns orchestration capabilities
	GetCapabilities() []string
}

// OrchestrationRequest represents a request to the orchestrator
type OrchestrationRequest struct {
	Message      string                  `json:"message"`
	Attachments  []entities.Attachment   `json:"attachments,omitempty"`
	MaxIterations int                     `json:"max_iterations"`
	Agent        string                  `json:"agent,omitempty"`
	ForceTool    string                  `json:"force_tool,omitempty"`
	Context      map[string]any          `json:"context,omitempty"`
}

// OrchestrationResponse represents a response from the orchestrator
type OrchestrationResponse struct {
	Reply        string                  `json:"reply"`
	Agent        string                  `json:"agent"`
	ToolCalls    []entities.ToolCall     `json:"tool_calls"`
	TokenUsage   TokenUsage              `json:"token_usage"`
	ExecTime     int64                   `json:"exec_time_ms"`
	Iterations   int                     `json:"iterations"`
	Error        string                  `json:"error,omitempty"`
}

// AgentRouter defines the interface for agent routing
type AgentRouter interface {
	// GetAgents returns all available agents
	GetAgents() []*entities.Agent

	// FindAgent finds an agent by ID
	FindAgent(id string) (*entities.Agent, error)

	// FindAgentByCapability finds agents by capability
	FindAgentByCapability(capability string) []*entities.Agent

	// RegisterAgent registers a new agent
	RegisterAgent(agent *entities.Agent) error

	// UnregisterAgent removes an agent
	UnregisterAgent(id string) error
}

// HandoffRequest represents a handoff request
type HandoffRequest struct {
	FromAgent    string                 `json:"from_agent"`
	ToAgent      string                 `json:"to_agent"`
	Reason       string                 `json:"reason"`
	Payload      map[string]any         `json:"payload"`
	Context      map[string]any         `json:"context,omitempty"`
}