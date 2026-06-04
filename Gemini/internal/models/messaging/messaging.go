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

type Message struct {
	Role             Role
	Content          string
	ToolCalls        []ToolCall
	ToolResponses    []ToolResponse
	ThoughtSignature string
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
