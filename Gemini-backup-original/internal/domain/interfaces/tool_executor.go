package interfaces

import (
	"context"
	"time"

	"gemini-cli/internal/domain/entities"
)

// ToolRegistry defines the interface for tool registry operations
type ToolRegistry interface {
	// Register registers a tool
	Register(tool *entities.Tool) error

	// GetTool returns a tool by name
	GetTool(name string) (*entities.Tool, error)

	// ListTools returns all registered tools
	ListTools() []*entities.Tool

	// GetTools returns tools for LLM function calling
	GetTools() []entities.ToolSchema

	// Unregister removes a tool
	Unregister(name string) error

	// Clear removes all tools
	Clear()
}

// ToolExecutor defines the interface for tool execution
type ToolExecutor interface {
	// Execute executes a tool
	Execute(ctx context.Context, req *ToolRequest) (*ToolResponse, error)

	// GetSchema returns the tool schema
	GetSchema() *entities.ToolSchema

	// Validate validates tool input
	Validate(args map[string]any) error

	// GetCapabilities returns tool capabilities
	GetCapabilities() []string
}

// ToolRequest represents a tool execution request
type ToolRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]any         `json:"arguments"`
	Context   *ToolContext           `json:"context"`
	Metadata  map[string]any         `json:"metadata,omitempty"`
}

// ToolResponse represents a tool execution response
type ToolResponse struct {
	Success    bool            `json:"success"`
	Data       interface{}     `json:"data"`
	Error      string          `json:"error,omitempty"`
	CacheHit   bool            `json:"cache_hit,omitempty"`
	ExecTime   time.Duration   `json:"exec_time,omitempty"`
	Metadata   map[string]any  `json:"metadata,omitempty"`
}

// ToolContext provides execution context for tools
type ToolContext struct {
	Context        map[string]any   `json:"context"`
	ConversationID string          `json:"conversation_id"`
	UserID         string          `json:"user_id"`
	Metadata       map[string]any   `json:"metadata"`
	Agent          *entities.Agent  `json:"agent"`
}

// ToolCapability represents tool capabilities
type ToolCapability struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Categories  []string `json:"categories"`
}