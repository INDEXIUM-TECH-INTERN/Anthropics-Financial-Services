package messaging

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type ToolCall struct {
	ID   string
	Name string
	Args map[string]interface{}
}

type ToolResponse struct {
	CallID  string
	Name    string
	Content string
}

type Attachment struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Data string `json:"data"` // Base64 encoded
}

type Message struct {
	Role             Role           `json:"role"`
	Content          string         `json:"content"`
	ToolCalls        []ToolCall     `json:"tool_calls,omitempty"`
	ToolResponses    []ToolResponse `json:"tool_responses,omitempty"`
	ThoughtSignature string         `json:"thought_signature,omitempty"`
	Attachments      []Attachment   `json:"attachments,omitempty"`
	LatencyMs        int64          `json:"latency_ms,omitempty"`
	TokenIn          int            `json:"token_in,omitempty"`
	TokenOut         int            `json:"token_out,omitempty"`
	RamMB            string         `json:"ram_mb,omitempty"`
	CpuLoad          string         `json:"cpu_load,omitempty"`
}

type ToolSchema struct {
	Name        string
	Description string
	Parameters  map[string]interface{}
}

type Request struct {
	History []Message
	Tools   []ToolSchema
}
