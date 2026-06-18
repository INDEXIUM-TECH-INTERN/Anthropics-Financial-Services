package entities

// Domain errors
var (
	ErrAgentNotFound      = &DomainError{Code: "AGENT_NOT_FOUND", Message: "Agent not found"}
	ErrToolNotFound      = &DomainError{Code: "TOOL_NOT_FOUND", Message: "Tool not found"}
	ErrInvalidInput      = &DomainError{Code: "INVALID_INPUT", Message: "Invalid input"}
	ErrConversationEmpty = &DomainError{Code: "CONVERSATION_EMPTY", Message: "Conversation is empty"}
	ErrMaxTokensExceeded = &DomainError{Code: "MAX_TOKENS_EXCEEDED", Message: "Max tokens exceeded"}
	ErrToolExecution     = &DomainError{Code: "TOOL_EXECUTION", Message: "Tool execution failed"}
)

// DomainError represents domain-level errors
type DomainError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *DomainError) Error() string {
	return e.Message
}

// IsDomainError checks if error is a domain error
func IsDomainError(err error) bool {
	_, ok := err.(*DomainError)
	return ok
}