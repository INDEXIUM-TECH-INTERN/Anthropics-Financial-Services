package models

// --- Groq Models (OpenAI Compatible) ---
type GroqRequest struct {
	Model       string        `json:"model"`
	Messages    []GroqMessage `json:"messages"`
	Tools       []GroqTool    `json:"tools,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
}

type GroqMessage struct {
	Role       string         `json:"role"`
	Content    string         `json:"content,omitempty"`
	ToolCalls  []GroqToolCall `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
}

type GroqTool struct {
	Type     string                  `json:"type"`
	Function GroqFunctionDeclaration `json:"function"`
}

type GroqFunctionDeclaration struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Parameters  Parameters `json:"parameters"`
}

type GroqToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function GroqFunction `json:"function"`
}

type GroqFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type GroqResponse struct {
	Choices []struct {
		Message GroqMessage `json:"message"`
	} `json:"choices"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}
