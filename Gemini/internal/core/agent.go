package core

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"gemini-cli/internal/models/messaging"
	"gemini-cli/internal/providers"
	"gemini-cli/internal/utils"
)

type Agent struct {
	mu           sync.Mutex
	provider     providers.Provider
	systemPrompt string
	userInput    string
	handoffPlan  *RoutePlan
	conversation *Conversation
	orchestrator *Orchestrator
	dispatcher   *Dispatcher
}

func NewAgent() *Agent {
	utils.LoadEnv()

	gemini := newGeminiProvider()

	model := os.Getenv("OPENROUTER_MODEL")
	if model == "" {
		model = "meta-llama/llama-3.3-70b-instruct:free"
	}

	var orProviders []providers.Provider
	keyNames := []string{"OPENROUTER_API_KEY", "OPENROUTER_API_KEY_2", "OPENROUTER_API_KEY_3"}
	for _, kn := range keyNames {
		val := os.Getenv(kn)
		if val != "" {
			orProviders = append(orProviders, &providers.OpenRouterProvider{
				APIKey: val,
				Model:  model,
			})
			fmt.Printf("🔑 [Config] Loaded OpenRouter Key from %s\n", kn)
		}
	}

	// Hỗ trợ bypass Gemini khi free tier quota hết hoặc muốn ưu tiên OpenRouter
	useOnlyOR := os.Getenv("USE_OPENROUTER_ONLY") == "1" || os.Getenv("GEMINI_API_KEY") == "" || os.Getenv("GEMINI_API_KEY") == "disabled"

	a := &Agent{
		systemPrompt: buildGroundedSystemPrompt(time.Now()),
		conversation: NewConversation("default"),
	}

	if useOnlyOR && len(orProviders) > 0 {
		fmt.Println("🚀 [Config] Sử dụng OpenRouter làm primary (bypass Gemini để tránh quota)")
		// Dùng OR đầu tiên làm primary, các key còn lại làm fallback
		primaryOR := orProviders[0]
		fallbackORs := orProviders[1:]
		a.provider = providers.NewMultiProvider(primaryOR, fallbackORs)
	} else {
		a.provider = providers.NewMultiProvider(gemini, orProviders)
	}

	a.orchestrator = NewOrchestrator(a)
	a.dispatcher = NewDispatcher(a)

	return a
}

func newGeminiProvider() *providers.GeminiProvider {
	return &providers.GeminiProvider{
		APIKey: os.Getenv("GEMINI_API_KEY"),
		Model:  normalizeGeminiModel(os.Getenv("GEMINI_MODEL")),
	}
}

func normalizeGeminiModel(model string) string {
	normalized := strings.TrimSpace(model)
	if normalized == "" {
		return "gemini-flash-latest"
	}
	if !strings.HasPrefix(normalized, "models/") {
		return "models/" + normalized
	}
	return normalized
}

func (a *Agent) SetOpenRouterKeys(keys []string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	model := os.Getenv("OPENROUTER_MODEL")
	if model == "" {
		model = "meta-llama/llama-3.3-70b-instruct:free"
	}

	var orProviders []providers.Provider
	for _, key := range keys {
		if strings.TrimSpace(key) == "" {
			continue
		}
		orProviders = append(orProviders, &providers.OpenRouterProvider{
			APIKey: key,
			Model:  model,
		})
	}

	gemini := newGeminiProvider()
	a.provider = providers.NewMultiProvider(gemini, orProviders)

	fmt.Printf("🔑 [Config] Updated OpenRouter keys. Count: %d\n", len(orProviders))
}

func buildGroundedSystemPrompt(now time.Time) string {
	return utils.RenderPromptTemplate("grounded_system_prompt.txt", map[string]string{
		"SYSTEM_TIME":        now.Format("02/01/2006 15:04:05"),
		"SYSTEM_WEEKDAY":     utils.TranslateWeekday(now.Weekday()),
		"BASE_SYSTEM_PROMPT": utils.LoadPrompt("system_prompt.txt"),
	})
}

func (a *Agent) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.conversation.Reset()
	a.handoffPlan = nil
	fmt.Println("🔄 [Agent] Conversation history reset.")
}

func (a *Agent) Start() {
	for {
		userInput, ok := readUserInput()
		if !ok {
			return
		}

		reply, err := a.ProcessMessage(userInput)
		if err != nil {
			fmt.Printf("❌ [Lỗi] %v\n", err)
			continue
		}
		_ = reply
	}
}

func (a *Agent) ProcessMessage(userInput string) (string, error) {
	return a.orchestrator.ProcessMessage(userInput)
}

func readUserInput() (string, bool) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("👤 Anthropic Financial Agent > ")
	if !scanner.Scan() {
		return "", false
	}

	return scanner.Text(), true
}

func (a *Agent) appendUserTextInternal(text string) {
	a.conversation.ContextWindow.History = append(a.conversation.ContextWindow.History, messaging.Message{
		Role:    messaging.RoleUser,
		Content: text,
	})
}

func (a *Agent) AddUserText(text string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.appendUserTextInternal(text)
}

func (a *Agent) GetHistory() []messaging.Message {
	a.mu.Lock()
	defer a.mu.Unlock()
	cp := make([]messaging.Message, len(a.conversation.ContextWindow.History))
	copy(cp, a.conversation.ContextWindow.History)
	return cp
}

// LoadHistory replaces the current conversation history (used for multi-session support with Redis).
func (a *Agent) LoadHistory(msgs []messaging.Message) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.conversation.ContextWindow.History = make([]messaging.Message, len(msgs))
	copy(a.conversation.ContextWindow.History, msgs)
}
