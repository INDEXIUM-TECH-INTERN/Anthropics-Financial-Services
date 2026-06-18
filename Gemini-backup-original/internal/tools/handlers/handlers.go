// Package handlers provides individual tool handler implementations.
// Each handler implements the ToolHandler interface for map-based dispatch.
package handlers

import "context"

// Args represents tool call arguments as a string-keyed map.
type Args = map[string]interface{}

// ToolHandler is the interface for all tool execution handlers.
type ToolHandler interface {
	// Name returns the tool's registered name (must match the LLM tool schema).
	Name() string
	// Description returns the tool's description for LLM tool schemas.
	Description() string
	// Execute runs the tool with the given arguments and returns a string result.
	Execute(ctx context.Context, args Args) (string, error)
}
