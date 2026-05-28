package models

// --- Gemini Models ---
type GeminiRequest struct {
	Contents          []GeminiContent `json:"contents"`
	Tools             []GeminiTool    `json:"tools,omitempty"`
	SystemInstruction *GeminiContent  `json:"systemInstruction,omitempty"`
	SafetySettings    []SafetySetting `json:"safetySettings,omitempty"`
}

type GeminiTool struct {
	FunctionDeclarations []GeminiFunctionDeclaration `json:"functionDeclarations,omitempty"`
}

type GeminiFunctionDeclaration struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

// type GeminiContent struct {
// 	Role  string       `json:"role"`
// 	Parts []GeminiPart `json:"parts"`
// }
type GeminiContent struct {
	Role  string       `json:"role"`
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text             string                  `json:"text,omitempty"`
	FunctionCall     *GeminiFunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *GeminiFunctionResponse `json:"functionResponse,omitempty"`
}

type GeminiFunctionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
	ID   string                 `json:"id,omitempty"`
}

type GeminiFunctionResponse struct {
	Name     string                 `json:"name"`
	Response map[string]interface{} `json:"response"`
	ID       string                 `json:"id,omitempty"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content GeminiContent `json:"content"`
	} `json:"candidates"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

type SafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}
