package entities

import (
	"encoding/json"
	"time"
)

// Tool represents a callable tool/function
type Tool struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Schema      *ToolSchema             `json:"schema"`
	Executor    ToolExecutor            `json:"-"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// ToolSchema defines the schema for a tool/function
type ToolSchema struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Required    []string               `json:"required"`
}

// ToolExecutor interface for tool execution
type ToolExecutor interface {
	Execute(ctx *ToolContext) (*ToolResult, error)
	GetSchema() *ToolSchema
	Validate(args map[string]any) error
}

// ToolContext provides execution context for a tool
type ToolContext struct {
	Context      map[string]any    `json:"context"`
	Input        map[string]any    `json:"input"`
	UserID       string            `json:"user_id"`
	ConversationID string          `json:"conversation_id"`
	Metadata     map[string]any    `json:"metadata"`
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Success    bool        `json:"success"`
	Data       interface{} `json:"data"`
	Error      string      `json:"error,omitempty"`
	CacheHit   bool        `json:"cache_hit,omitempty"`
	ExecTime   time.Duration `json:"exec_time,omitempty"`
}

// NewTool creates a new tool instance
func NewTool(id, name, description string, schema *ToolSchema) *Tool {
	return &Tool{
		ID:          id,
		Name:        name,
		Description: description,
		Schema:      schema,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// SetExecutor sets the tool executor
func (t *Tool) SetExecutor(executor ToolExecutor) {
	t.Executor = executor
	t.UpdatedAt = time.Now()
}

// Execute executes the tool
func (t *Tool) Execute(ctx *ToolContext) (*ToolResult, error) {
	if t.Executor == nil {
		return nil, &ToolError{Code: "NO_EXECUTOR", Message: "Tool executor not set"}
	}

	start := time.Now()
	result, err := t.Executor.Execute(ctx)
	result.ExecTime = time.Since(start)

	return result, err
}

// MarshalJSON custom marshal for Tool
func (t *Tool) MarshalJSON() ([]byte, error) {
	type Alias Tool
	return json.Marshal(&struct {
		Executor interface{} `json:"executor"`
		*Alias
	}{
		Executor: nil, // Don't serialize executor
		Alias:    (*Alias)(t),
	})
}

// ToolError represents tool execution errors
type ToolError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *ToolError) Error() string {
	return e.Message
}