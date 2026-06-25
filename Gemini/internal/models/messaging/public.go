package messaging

import (
	"regexp"
	"strings"
)

var bootstrapBlockRe = regexp.MustCompile(`(?is)ANTHROPIC AGENT CONFIGURATION[\s\S]*?(?:SKILL MARKDOWN \([^)]+\)[\s\S]*?)(?:\n\n|$)`)

var responseStartMarkers = []string{
	"Chào bạn",
	"# Báo cáo",
	"## Báo cáo",
	"### Báo cáo",
}

var skillLeakMarkers = []string{
	"### Step ",
	"## Important Notes",
	"SKILL MARKDOWN (",
	"SYSTEM PROMPT (from agents/",
}

// FilterPublicHistory returns messages safe to show in the chat UI.
func FilterPublicHistory(history []Message) []Message {
	if len(history) == 0 {
		return history
	}
	out := make([]Message, 0, len(history))
	for _, msg := range history {
		if msg.Internal || IsBootstrapPayload(msg.Content) {
			continue
		}
		if msg.Role == RoleAssistant {
			msg.Content = SanitizeAssistantContent(msg.Content)
		}
		out = append(out, msg)
	}
	return out
}

// IsBootstrapPayload detects agent/skill bootstrap blobs that must stay hidden.
func IsBootstrapPayload(content string) bool {
	lower := strings.ToLower(content)
	markers := []string{
		"anthropic agent configuration",
		"system prompt (from agents/",
		"skill markdown (",
	}
	for _, marker := range markers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

// SanitizeAssistantContent removes leaked skill/bootstrap instructions from model output.
func SanitizeAssistantContent(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return text
	}

	if start := findResponseStart(text); start > 0 && hasSkillLeakBefore(text[:start]) {
		text = strings.TrimSpace(text[start:])
	}

	text = bootstrapBlockRe.ReplaceAllString(text, "")
	return strings.TrimSpace(text)
}

func hasSkillLeakBefore(prefix string) bool {
	for _, marker := range skillLeakMarkers {
		if strings.Contains(prefix, marker) {
			return true
		}
	}
	return false
}

func findResponseStart(text string) int {
	best := -1
	for _, marker := range responseStartMarkers {
		if idx := strings.Index(text, marker); idx >= 0 {
			if best < 0 || idx < best {
				best = idx
			}
		}
	}
	return best
}