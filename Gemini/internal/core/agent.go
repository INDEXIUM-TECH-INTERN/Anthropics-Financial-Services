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
	requestMu    sync.Mutex // serialises concurrent HTTP requests using the same agent
}

func NewAgent() *Agent {
	utils.LoadEnv()

	geminiProviders := newGeminiProviders()
	orProviders := newOpenRouterProviders(openRouterKeysFromEnv())

	// Hỗ trợ bypass Gemini khi free tier quota hết hoặc muốn ưu tiên OpenRouter
	useOnlyOR := os.Getenv("USE_OPENROUTER_ONLY") == "1" || len(geminiProviders) == 0

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
		allProviders := append(geminiProviders, orProviders...)
		if len(allProviders) == 0 {
			allProviders = append(allProviders, newGeminiProvider(""))
		}
		a.provider = providers.NewMultiProvider(allProviders[0], allProviders[1:])
	}

	a.orchestrator = NewOrchestrator(a)
	a.dispatcher = NewDispatcher(a)

	return a
}

func numberedEnvKeys(base string, max int) []string {
	keys := []string{base}
	for i := 2; i <= max; i++ {
		keys = append(keys, fmt.Sprintf("%s_%d", base, i))
	}
	return keys
}

func envValues(keys []string) []string {
	var values []string
	for _, key := range keys {
		value := strings.TrimSpace(os.Getenv(key))
		if value == "" || value == "disabled" {
			continue
		}
		values = append(values, value)
		fmt.Printf("🔑 [Config] Loaded key from %s\n", key)
	}
	return values
}

func openRouterKeysFromEnv() []string {
	return envValues(numberedEnvKeys("OPENROUTER_API_KEY", 5))
}

func newGeminiProvider(apiKey string) *providers.GeminiProvider {
	return &providers.GeminiProvider{
		APIKey: apiKey,
		Model:  normalizeGeminiModel(os.Getenv("GEMINI_MODEL")),
	}
}

func newGeminiProviders() []providers.Provider {
	keys := envValues(numberedEnvKeys("GEMINI_API_KEY", 5))
	geminis := make([]providers.Provider, 0, len(keys))
	for _, key := range keys {
		geminis = append(geminis, newGeminiProvider(key))
	}
	return geminis
}

func newOpenRouterProviders(keys []string) []providers.Provider {
	model := os.Getenv("OPENROUTER_MODEL")
	if model == "" {
		model = "meta-llama/llama-3.3-70b-instruct:free"
	}

	openRouters := make([]providers.Provider, 0, len(keys))
	for _, key := range keys {
		openRouters = append(openRouters, &providers.OpenRouterProvider{
			APIKey: key,
			Model:  model,
		})
	}
	return openRouters
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

	var cleanKeys []string
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		cleanKeys = append(cleanKeys, key)
	}

	geminiProviders := newGeminiProviders()
	orProviders := newOpenRouterProviders(cleanKeys)
	allProviders := append(geminiProviders, orProviders...)
	if len(allProviders) == 0 {
		allProviders = append(allProviders, newGeminiProvider(""))
	}
	a.provider = providers.NewMultiProvider(allProviders[0], allProviders[1:])

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

		reply, err := a.ProcessMessage(userInput, nil)
		if err != nil {
			fmt.Printf("❌ [Lỗi] %v\n", err)
			continue
		}
		fmt.Printf("🤖 Agent: %s\n", reply)
	}
}

func (a *Agent) ProcessMessage(userInput string, atts []messaging.Attachment) (string, error) {
	a.requestMu.Lock()
	defer a.requestMu.Unlock()
	return a.orchestrator.ProcessMessage(userInput, atts)
}

// ProcessMessageStream processes a message and streams tokens via onChunk.
// The final chunk has Done=true and the full text in Text.
func (a *Agent) ProcessMessageStream(userInput string, atts []messaging.Attachment, onChunk func(string, bool)) error {
	a.requestMu.Lock()
	defer a.requestMu.Unlock()
	return a.orchestrator.ProcessMessageStream(userInput, atts, onChunk)
}

func readUserInput() (string, bool) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("👤 Anthropic Financial Agent > ")
	if !scanner.Scan() {
		return "", false
	}

	return scanner.Text(), true
}

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

			// Tự động parse nội dung nếu là file text/excel (R6.2)
			if content, ok := utils.ParseAttachment(at.Name, at.Type, at.Data); ok {
				fmt.Printf("📄 [Parser] Tự động trích xuất nội dung từ file: %s\n", at.Name)
				parsedContents = append(parsedContents, utils.GetFileContentWrapper(at.Name, content))
			} else if content != "" {
				fmt.Printf("⚠️ [Parser] Không thể trích xuất nội dung từ file %s: %s\n", at.Name, content)
				// Nếu không parse được nhưng có thông báo lỗi, vẫn có thể cân nhắc append thông báo (tùy chọn)
				// Ở đây ta giữ nguyên hành vi chỉ log.
			}
		}

		if len(parsedContents) > 0 {
			msg.Content += "\n\n" + strings.Join(parsedContents, "\n\n")
		}
	}
	a.conversation.ContextWindow.History = append(a.conversation.ContextWindow.History, msg)
}

func (a *Agent) AddUserText(text string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.appendUserTextInternal(text, nil)
}

func (a *Agent) GetHistory() []messaging.Message {
	a.mu.Lock()
	defer a.mu.Unlock()
	cp := make([]messaging.Message, len(a.conversation.ContextWindow.History))
	copy(cp, a.conversation.ContextWindow.History)
	return cp
}

// GetProvider returns the current provider safely under the mutex.
// This prevents data races when SetOpenRouterKeys is called concurrently.
func (a *Agent) GetProvider() providers.Provider {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.provider
}

// LoadHistory replaces the current conversation history (used for multi-session support with Redis).
func (a *Agent) LoadHistory(msgs []messaging.Message) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.conversation.ContextWindow.History = make([]messaging.Message, len(msgs))
	copy(a.conversation.ContextWindow.History, msgs)
}
