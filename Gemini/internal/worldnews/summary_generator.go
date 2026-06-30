package worldnews

// TextGenerator generates short text (e.g. Gemini).
type TextGenerator interface {
	GenerateText(systemPrompt, userPrompt string) (string, error)
}