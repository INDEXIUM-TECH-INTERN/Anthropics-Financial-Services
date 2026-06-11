// Package errors provides domain-specific error types for the backend.
package errors

import "fmt"

// ErrProviderFailure indicates all LLM providers failed (quota/rate-limit/connection).
type ErrProviderFailure struct {
	Err error
}

func (e *ErrProviderFailure) Error() string {
	return fmt.Sprintf("Tat ca cac dich vu AI deu that bai: %v", e.Err)
}

func (e *ErrProviderFailure) Unwrap() error { return e.Err }

// ErrRoutingFailure indicates the router failed to select a valid agent.
type ErrRoutingFailure struct {
	Query string
	Reason string
}

func (e *ErrRoutingFailure) Error() string {
	return fmt.Sprintf("Routing failed for query '%s ...': %s", truncate(e.Query, 40), e.Reason)
}

// ErrContextOverflow indicates the context window exceeded the token limit.
type ErrContextOverflow struct {
	Tokens  int
	MaxTokens int
}

func (e *ErrContextOverflow) Error() string {
	return fmt.Sprintf("Context overflow: %d tokens exceeds limit of %d", e.Tokens, e.MaxTokens)
}

// ErrSessionNotFound indicates a chat session was not found in storage.
type ErrSessionNotFound struct {
	SessionID string
}

func (e *ErrSessionNotFound) Error() string {
	return fmt.Sprintf("Session not found: %s", e.SessionID)
}

// ErrToolExecution indicates a tool failed during execution.
type ErrToolExecution struct {
	ToolName string
	Err      error
}

func (e *ErrToolExecution) Error() string {
	return fmt.Sprintf("Tool '%s' execution failed: %v", e.ToolName, e.Err)
}

func (e *ErrToolExecution) Unwrap() error { return e.Err }

// NewProviderFailure wraps an error as ErrProviderFailure.
func NewProviderFailure(err error) error {
	return &ErrProviderFailure{Err: err}
}

// NewRoutingFailure creates a new ErrRoutingFailure.
func NewRoutingFailure(query, reason string) error {
	return &ErrRoutingFailure{Query: query, Reason: reason}
}

// NewContextOverflow creates a new ErrContextOverflow.
func NewContextOverflow(tokens, max int) error {
	return &ErrContextOverflow{Tokens: tokens, MaxTokens: max}
}

// NewSessionNotFound creates a new ErrSessionNotFound.
func NewSessionNotFound(id string) error {
	return &ErrSessionNotFound{SessionID: id}
}

// NewToolExecution creates a new ErrToolExecution.
func NewToolExecution(name string, err error) error {
	return &ErrToolExecution{ToolName: name, Err: err}
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
