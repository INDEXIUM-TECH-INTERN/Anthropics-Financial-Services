package models

// OpenRouter uses OpenAI-compatible structures. 
// We can reuse Groq models or define specific ones if we want to handle OpenRouter-specific fields like 'transforms'.

type OpenRouterRequest struct {
	Model    string          `json:"model"`
	Messages []OpenRouterMessage `json:"messages"`
	Tools    []OpenRouterTool    `json:"tools,omitempty"`
}

type OpenRouterMessage struct {
	Role       string               `json:"role"`
	Content    string               `json:"content,omitempty"`
	ToolCalls  []OpenRouterToolCall `json:"tool_calls,omitempty"`
	ToolCallID string               `json:"tool_call_id,omitempty"`
}

type OpenRouterTool struct {
	Type     string                        `json:"type"`
	Function OpenRouterFunctionDeclaration `json:"function"`
}

type OpenRouterFunctionDeclaration struct {
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
