package core

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"gemini-cli/internal/models/messaging"
	"gemini-cli/internal/providers"
	"gemini-cli/internal/utils"
)

// Agent is the central facade that wires all subsystems together.
// It is safe for concurrent use: all shared state is protected by mu (sync.RWMutex).
// The mutex serializes both HTTP request processing and internal state access,
// eliminating the previous deadlock risk from nested mu + requestMu locking.
type Agent struct {
	mu           sync.RWMutex
	pm           *ProviderManager
	systemPrompt string
	userInput    string
	handoffPlan  *RoutePlan
	conversation *Conversation
	orchestrator *Orchestrator
	dispatcher   *Dispatcher
}

func NewAgent() *Agent {
	utils.LoadEnv()

	a := &Agent{
		pm:           NewProviderManager(),
		systemPrompt: buildGroundedSystemPrompt(time.Now()),
		conversation: NewConversation("default"),
	}

	a.orchestrator = NewOrchestrator(a)
	a.dispatcher = NewDispatcher(a)

	return a
}

// SetOpenRouterKeys updates the provider chain at runtime with new OpenRouter keys.
func (a *Agent) SetOpenRouterKeys(keys []string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pm.SetOpenRouterKeys(keys)
}

func buildGroundedSystemPrompt(now time.Time) string {
	currentYear := now.Year()
	return utils.RenderPromptTemplate("grounded_system_prompt.txt", map[string]string{
		"SYSTEM_TIME":        now.Format("02/01/2006 15:04:05"),
		"SYSTEM_WEEKDAY":     utils.TranslateWeekday(now.Weekday()),
		"CURRENT_YEAR":       fmt.Sprintf("%d", currentYear),
		"YEAR_MINUS_1":       fmt.Sprintf("%d", currentYear-1),
		"YEAR_MINUS_2":       fmt.Sprintf("%d", currentYear-2),
		"BASE_SYSTEM_PROMPT": utils.LoadPrompt("system_prompt.txt"),
	})
}

// Reset clears the conversation history and handoff state.
func (a *Agent) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.conversation.Reset()
	a.handoffPlan = nil
	fmt.Println("🔄 [Agent] Conversation history reset.")
}

// Start runs the interactive CLI loop.
func (a *Agent) Start() {
	for {
		userInput, ok := readUserInput()
		if !ok {
			return
		}

		reply, err := a.ProcessMessage(context.Background(), userInput, nil)
		if err != nil {
			fmt.Printf("❌ [Lỗi] %v\n", err)
			continue
		}
		fmt.Printf("🤖 Agent: %s\n", reply)
	}
}

// ProcessMessage processes a user message and returns the AI response.
// Locking is managed internally by the orchestrator for minimal critical sections.
func (a *Agent) ProcessMessage(ctx context.Context, userInput string, atts []messaging.Attachment) (string, error) {
	return a.orchestrator.ProcessMessage(ctx, userInput, atts)
}

// ProcessMessageStream processes a message and streams tokens via onChunk.
// Locking is managed internally by the orchestrator for minimal critical sections.
func (a *Agent) ProcessMessageStream(ctx context.Context, userInput string, atts []messaging.Attachment, onChunk func(string, bool)) error {
	return a.orchestrator.ProcessMessageStream(ctx, userInput, atts, onChunk)
}

func readUserInput() (string, bool) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("👤 Anthropic Financial Agent > ")
	if !scanner.Scan() {
		return "", false
	}
	return scanner.Text(), true
}

// appendUserTextInternal appends a user message to the conversation history.
// Must be called with a.mu held (write lock).
func (a *Agent) appendUserTextInternal(text string, atts []messaging.Attachment) {
	msg := messaging.Message{
		Role:    messaging.RoleUser,
		Content: text,
	}
	if len(atts) > 0 {
		msg.Attachments = make([]messaging.Attachment, len(atts))
		var parsedContents []string

		for i, at := range atts {
			msg.Attachments[i] = messaging.Attachment{
				Name: at.Name,
				Type: at.Type,
				Data: at.Data,
			}

			if content, ok := utils.ParseAttachment(at.Name, at.Type, at.Data); ok {
				fmt.Printf("📄 [Parser] Tự động trích xuất nội dung từ file: %s\n", at.Name)
				parsedContents = append(parsedContents, utils.GetFileContentWrapper(at.Name, content))
			} else if content != "" {
				fmt.Printf("⚠️ [Parser] Không thể trích xuất nội dung từ file %s: %s\n", at.Name, content)
			}
		}

		if len(parsedContents) > 0 {
			msg.Content += "\n\n" + strings.Join(parsedContents, "\n\n")
		}
	}
	a.conversation.ContextWindow.History = append(a.conversation.ContextWindow.History, msg)
}

// GetHistory returns a copy of the full conversation history.
func (a *Agent) GetHistory() []messaging.Message {
	a.mu.RLock()
	defer a.mu.RUnlock()
	cp := make([]messaging.Message, len(a.conversation.ContextWindow.History))
	copy(cp, a.conversation.ContextWindow.History)
	return cp
}

// GetProvider returns the current provider safely under the mutex.
func (a *Agent) GetProvider() providers.Provider {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.pm.GetProvider()
}

// LoadHistory replaces the current conversation history (used for multi-session support with Redis).
func (a *Agent) LoadHistory(msgs []messaging.Message) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.conversation.ContextWindow.History = make([]messaging.Message, len(msgs))
	copy(a.conversation.ContextWindow.History, msgs)
}
