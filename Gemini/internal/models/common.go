package models

// --- Common Shared Models ---
type Parameters struct {
	Name        string              `json:"-"`
	Description string              `json:"-"`
	Type        string              `json:"type"`
	Properties  map[string]Property `json:"properties"`
	Required    []string            `json:"required"`
}

type Property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// --- OpenRouter / OpenAI Compatible Models ---
type OpenRouterRequest struct {
	Model       string              `json:"model"`
	Messages    []OpenRouterMessage `json:"messages"`
	Tools       []OpenRouterTool    `json:"tools,omitempty"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
	Temperature float64             `json:"temperature,omitempty"`
}

type OpenRouterMessage struct {
	Role       string               `json:"role"`
	Content    string               `json:"content,omitempty"`
	ToolCalls  []OpenRouterToolCall `json:"tool_calls,omitempty"`
	ToolCallID string               `json:"tool_call_id,omitempty"`
}

type OpenRouterTool struct {
	Type     string                    `json:"type"`
	Function OpenRouterFunctionDeclare `json:"function"`
}

type OpenRouterFunctionDeclare struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Parameters  Parameters `json:"parameters"`
}

type OpenRouterToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function OpenRouterFunction `json:"function"`
}

type OpenRouterFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type OpenRouterResponse struct {
	Choices []struct {
		Message OpenRouterMessage `json:"message"`
	} `json:"choices"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}
