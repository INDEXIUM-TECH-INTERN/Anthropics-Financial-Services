package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"gemini-cli/internal/models"
	"gemini-cli/internal/providers"
	"gemini-cli/internal/tools"
	"gemini-cli/internal/utils"
)

type Agent struct {
	mu                  sync.Mutex
	geminiProvider      *providers.GeminiProvider
	openrouterProviders []*providers.OpenRouterProvider
	currentORIdx        int
	history             []models.GeminiContent
	systemPrompt        string
	tools               []models.Parameters
	userInput           string
	handoffPlan         *RoutePlan
	conversation        *Conversation

	// Quota tracking for Gemini
	geminiStrikes       int
	geminiCooldownUntil time.Time
}

func NewAgent() *Agent {
	utils.LoadEnv()

	return &Agent{
		geminiProvider:    newGeminiProvider(),
		groqProvider:      newGroqProvider(),
		sambanovaProvider: newSambanovaProvider(),
		systemPrompt:      buildGroundedSystemPrompt(time.Now()),
		conversation:      NewConversation("default"),
		tools: []models.Parameters{
			newFinancialResearchTool(),
			newFinancialScrapeTool(),
			newFinancialCalculateTool(),
			newHandoffRequestTool(),
			newLoadContextTool(),
		},
	}
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

func NewAgent() *Agent {
	utils.LoadEnv()

	a := &Agent{
		geminiProvider: newGeminiProvider(),
		history:        []models.GeminiContent{},
		systemPrompt:   buildGroundedSystemPrompt(time.Now()),
		tools: []models.Parameters{
			newFinancialResearchTool(),
			newFinancialScrapeTool(),
			newFinancialCalculateTool(),
			newHandoffRequestTool(),
			newLoadContextTool(),
		},
	}
	
	// Initialize with all environment keys (OPENROUTER_API_KEY, OPENROUTER_API_KEY_2, etc.)
	model := os.Getenv("OPENROUTER_MODEL")
	if model == "" { model = "openrouter/free" }

	keyNames := []string{"OPENROUTER_API_KEY", "OPENROUTER_API_KEY_2", "OPENROUTER_API_KEY_3"}
	for _, kn := range keyNames {
		val := os.Getenv(kn)
		if val != "" {
			a.openrouterProviders = append(a.openrouterProviders, &providers.OpenRouterProvider{
				APIKey: val,
				Model:  model,
			})
			fmt.Printf("🔑 [Config] Loaded OpenRouter Key from %s\n", kn)
		}
	}
	
	return a
}

func (a *Agent) SetOpenRouterKeys(keys []string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	a.openrouterProviders = nil
	model := os.Getenv("OPENROUTER_MODEL")
	if model == "" { model = "openrouter/free" }
	
	for _, key := range keys {
		if strings.TrimSpace(key) == "" { continue }
		a.openrouterProviders = append(a.openrouterProviders, &providers.OpenRouterProvider{
			APIKey: key,
			Model:  model,
		})
	}
	a.currentORIdx = 0
	fmt.Printf("🔑 [Config] Updated OpenRouter keys. Count: %d\n", len(a.openrouterProviders))
}

func (a *Agent) getNextOpenRouter() *providers.OpenRouterProvider {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	if len(a.openrouterProviders) == 0 {
		return nil
	}
	
	p := a.openrouterProviders[a.currentORIdx]
	a.currentORIdx = (a.currentORIdx + 1) % len(a.openrouterProviders)
	return p
}

// --- MCP Styled Tools ---

func newFinancialResearchTool() models.Parameters {
	return models.Parameters{
		Name:        "financial_research",
		Description: "Truy vấn dữ liệu thị trường thực tế (tương đương MCP Data Connectors).",
		Type:        "object",
		Properties: map[string]models.Property{
			"query": {Type: "string", Description: "Từ khóa hoặc mã chứng khoán cần tra cứu"},
		},
		Required: []string{"query"},
	}
}

func newFinancialScrapeTool() models.Parameters {
	return models.Parameters{
		Name:        "financial_scrape",
		Description: "Đọc sâu nội dung báo cáo, tin tức từ URL (tương đương Web/PDF Reader MCP).",
		Type:        "object",
		Properties: map[string]models.Property{
			"url": {Type: "string", Description: "URL nguồn tài liệu"},
		},
		Required: []string{"url"},
	}
}

func newFinancialCalculateTool() models.Parameters {
	return models.Parameters{
		Name:        "financial_calculate",
		Description: "Thực hiện tính toán tài chính (tương đương Python/Excel tools).",
		Type:        "object",
		Properties: map[string]models.Property{
			"expression": {Type: "string", Description: "Biểu thức toán học (ví dụ: DCF, P/E calculation)"},
		},
		Required: []string{"expression"},
	}
}

func newHandoffRequestTool() models.Parameters {
	return models.Parameters{
		Name:        "handoff_request",
		Description: "Gửi yêu cầu chuyển giao tác vụ cho Agent chuyên môn khác (tương đương orchestrate.py).",
		Type:        "object",
		Properties: map[string]models.Property{
			"target_agent": {Type: "string", Description: "Agent mục tiêu (ví dụ: earnings-reviewer, model-builder)"},
			"reason":       {Type: "string", Description: "Lý do và ngữ cảnh cần chuyển giao"},
			"task_payload": {Type: "string", Description: "Dữ liệu/Nhiệm vụ cụ thể cần thực hiện"},
		},
		Required: []string{"target_agent", "reason"},
	}
}

func newLoadContextTool() models.Parameters {
	return models.Parameters{
		Name:        "load_financial_context",
		Description: "Nạp tài liệu kỹ năng (Skill) từ bộ Vertical Plugins.",
		Type:        "object",
		Properties: map[string]models.Property{
			"type": {Type: "string", Description: "agent hoặc skill"},
			"name": {Type: "string", Description: "Tên tài liệu (e.g. market-researcher/sector-overview)"},
		},
		Required: []string{"type", "name"},
	}
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
	a.mu.Lock()
	a.userInput = userInput

	isNewConversation := len(a.conversation.ContextWindow.History) == 0

	if isNewConversation {
		BroadcastLog("Khởi tạo cuộc hội thoại mới...", "process")
		if strings.HasPrefix(userInput, "/") {
			if a.handleSlashCommandInternal(userInput) {
				// handled
			}
		} else {
			a.appendUserTextInternal(userInput)
			a.bootstrapContextInternal()
		}
	} else {
		a.appendUserTextInternal(userInput)
	}
	a.mu.Unlock()

	return a.runConversationLoopInternal()
}

func (a *Agent) handleSlashCommandInternal(input string) bool {
	parts := strings.Fields(input)
	cmd := strings.ToLower(parts[0])
	args := strings.Join(parts[1:], " ")

	var route RoutePlan
	switch cmd {
	case "/earnings":
		BroadcastLog("Kích hoạt lệnh /earnings...", "routing")
		route = RoutePlan{Agent: "earnings-reviewer", Skills: []string{"earnings-analysis"}, Reason: "Slash command /earnings"}
	case "/market":
		BroadcastLog("Kích hoạt lệnh /market...", "routing")
		route = RoutePlan{Agent: "market-researcher", Skills: []string{"sector-overview"}, Reason: "Slash command /market"}
	case "/help":
		fmt.Println("\n-- ANTHROPIC CLI SIMULATOR COMMANDS --")
		fmt.Println("/earnings <ticker> : Run Earnings Reviewer workflow")
		fmt.Println("/market <query>   : Run Market Researcher workflow")
		return false
	default:
		return false
	}

	a.userInput = args
	a.appendUserTextInternal(args)
	a.executeBootstrapWithRouteInternal(route)
	return true
}

func (a *Agent) executeBootstrapWithRouteInternal(route RoutePlan) {
	BroadcastLog(fmt.Sprintf("Nạp cấu hình cho Agent: %s...", route.Agent), "routing")
	fmt.Printf("🧭 [Context] Orchestrator: Loading %s configuration...\n", route.Agent)
	contextParts := a.buildBootstrapContextInternal(route)
	bootstrapPayload := strings.Join(contextParts, "\n\n")
	a.appendUserTextInternal(bootstrapPayload)
}

func readUserInput() (string, bool) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("\n👤 Anthropic Financial Agent > ")
	if !scanner.Scan() {
		return "", false
	}

	return scanner.Text(), true
}

func (a *Agent) appendUserTextInternal(text string) {
	a.conversation.ContextWindow.History = append(a.conversation.ContextWindow.History, models.GeminiContent{
		Role:  "user",
		Parts: []models.GeminiPart{{Text: text}},
	})
}

// Exported for CLI startup context only
func (a *Agent) AddUserText(text string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.appendUserTextInternal(text)
}

func (a *Agent) runConversationLoopInternal() (string, error) {
	for {
		a.mu.Lock()
		systemPrompt := a.systemPrompt
		history := make([]models.GeminiContent, len(a.history))
		copy(history, a.history)
		tools := a.tools
		a.mu.Unlock()

		aiMessage, err := a.executeModelCall(systemPrompt, history, tools)
		if err != nil {
			return "", err
		}

<<<<<<< HEAD
		a.mu.Lock()
		a.history = append(a.history, aiMessage)
=======
		a.conversation.ContextWindow.History = append(a.conversation.ContextWindow.History, aiMessage)
>>>>>>> 7b74a62 (Add context window and conversation management)
		hasToolCall := a.handleToolCalls(aiMessage)

		if a.handoffPlan != nil {
			plan := *a.handoffPlan
			a.handoffPlan = nil
			fmt.Printf("\n🔀 [Orchestrator] Executing handoff to: %s\n", plan.Agent)
			a.executeBootstrapWithRouteInternal(plan)
			a.mu.Unlock()
			continue
		}
		a.mu.Unlock()

		if !hasToolCall {
			return extractResponseText(aiMessage), nil
		}
	}
}

func (a *Agent) executeModelCall(systemPrompt string, history []models.GeminiContent, tools []models.Parameters) (models.GeminiContent, error) {
	// Try Gemini first
	msg, err := a.callGeminiWithLog(systemPrompt, history, tools)
	if err == nil {
		return msg, nil
	}

	// Fallback to OpenRouter rotation
	return a.callOpenRouterWithFallback(systemPrompt, history, tools)
}

func (a *Agent) callGeminiWithLog(systemPrompt string, history []models.GeminiContent, tools []models.Parameters) (models.GeminiContent, error) {
	BroadcastLog("Đang gọi mô hình chính (Gemini)...", "process")
	aiMessage, err := a.geminiProvider.Call(systemPrompt, history, tools...)
	if err == nil {
		a.mu.Lock()
		a.geminiStrikes = 0
		a.mu.Unlock()
		return aiMessage, nil
	}

	BroadcastLog(fmt.Sprintf("Gemini lỗi: %v", err), "error")
	fmt.Printf("⚠️ [Fallback] Gemini lỗi: %v\n", err)
	return aiMessage, err
}

func (a *Agent) callOpenRouterWithFallback(systemPrompt string, history []models.GeminiContent, tools []models.Parameters) (models.GeminiContent, error) {
	numKeys := len(a.openrouterProviders)
	if numKeys == 0 {
		return models.GeminiContent{}, fmt.Errorf("gemini lỗi và không có OpenRouter keys nào được cấu hình để fallback")
	}

	var lastErr error
	for i := 0; i < numKeys; i++ {
		a.mu.Lock()
		activeIdx := a.currentORIdx
		a.mu.Unlock()

		nextOR := a.getNextOpenRouter()
		if nextOR == nil {
			continue
		}

		BroadcastLog(fmt.Sprintf("Đang sử dụng mô hình dự phòng (OpenRouter - Key #%d)...", activeIdx+1), "process")
		fmt.Printf("🔄 [Fallback] Đang thử OpenRouter Key #%d...\n", activeIdx+1)

		aiMessage, err := nextOR.Call(systemPrompt, history, tools...)
		if err == nil {
			return aiMessage, nil
		}

		lastErr = err
		BroadcastLog(fmt.Sprintf("OpenRouter Key #%d lỗi: %v", activeIdx+1, err), "error")
		fmt.Printf("⚠️ [Fallback] OpenRouter Key #%d lỗi: %v\n", activeIdx+1, err)
	}

	return models.GeminiContent{}, fmt.Errorf("tất cả %d OpenRouter keys cấu hình đều thất bại: %v", numKeys, lastErr)
}

func extractResponseText(aiMessage models.GeminiContent) string {
	for i := len(aiMessage.Parts) - 1; i >= 0; i-- {
		if aiMessage.Parts[i].Text != "" {
			return aiMessage.Parts[i].Text
		}
	}
	return ""
}

func (a *Agent) bootstrapContextInternal() {
	route := a.selectRoutePlan()
	fmt.Printf("🧭 [Router] Identified Agent: %s (Reason: %s)\n", route.Agent, route.Reason)
	a.executeBootstrapWithRouteInternal(route)
}

func (a *Agent) buildBootstrapContextInternal(route RoutePlan) []string {
	agentDoc := tools.LoadDocumentWithMetadata("agent", route.Agent)
	a.logLoadedDocument(agentDoc)

	contextParts := []string{
		fmt.Sprintf("ANTHROPIC AGENT CONFIGURATION\nAgent: %s\nSkills: %s\nMode: Managed Agent (API)", route.Agent, strings.Join(route.Skills, ", ")),
		fmt.Sprintf("SYSTEM PROMPT (from agents/%s.md)\n%s", route.Agent, agentDoc.Content),
	}

	// Chỉ nạp tối đa 1 skill đầu tiên để tiết kiệm token và tránh lỗi Request too large
	maxSkills := 1
	for i, skill := range route.Skills {
		if i >= maxSkills {
			fmt.Printf("⚠️ [Context] Bỏ qua skill %s để tối ưu hóa token.\n", skill)
			continue
		}
		BroadcastLog(fmt.Sprintf("Đang nạp skill chuyên biệt: %s", skill), "process")
		skillDoc := tools.LoadDocumentWithMetadata("skill", route.Agent+"/"+skill)
		a.logLoadedDocument(skillDoc)

		content := skillDoc.Content
		if len(content) > 4000 {
			content = content[:4000] + "\n... [Nội dung bị cắt bớt để tối ưu hóa context]"
		}
		contextParts = append(contextParts, fmt.Sprintf("SKILL MARKDOWN (%s)\n%s", skill, content))
	}

	if tools.NeedsRealtimeData(a.userInput) {
		BroadcastLog("Phát hiện nhu cầu dữ liệu Real-time. Đang tìm kiếm...", "process")
		queryPlan := tools.BuildMarketQueryPlan(a.userInput)
		realtimeResult := tools.SearchGoogle(queryPlan.SearchQuery)
		contextParts = append(contextParts, fmt.Sprintf("REAL-TIME MARKET DATA\n%s", realtimeResult))
	}

	contextParts = append(contextParts, utils.LoadPrompt("bootstrap_context_suffix.txt"))
	return contextParts
}

<<<<<<< HEAD
=======
func (a *Agent) askProvidersInternal() (models.GeminiContent, error) {
	// 1. Gemini (Primary)
	aiMessage, err := a.geminiProvider.Call(a.systemPrompt, a.conversation.ContextWindow.History, a.tools...)
	if err == nil {
		return aiMessage, nil
	}
	fmt.Printf("⚠️ [Fallback] Gemini lỗi: %v\n", err)

	// 2. SambaNova (Backup 1 - High Rate Limit Free)
	aiMessage, err = a.sambanovaProvider.Call(a.systemPrompt, a.conversation.ContextWindow.History, a.tools...)
	if err == nil {
		return aiMessage, nil
	}
	fmt.Printf("⚠️ [Fallback] SambaNova lỗi: %v\n", err)

	// 3. Groq (Backup 2)
	aiMessage, err = a.groqProvider.Call(a.systemPrompt, a.conversation.ContextWindow.History, a.tools...)
	if err != nil {
		if strings.Contains(err.Error(), "Rate limit reached") {
			fmt.Println("⏳ [Hệ thống] Chạm giới hạn Groq TPM. Đang chờ 15s để thử lại...")
			time.Sleep(15 * time.Second)
			aiMessage, err = a.groqProvider.Call(a.systemPrompt, a.conversation.ContextWindow.History, a.tools...)
		}
	}
	if err == nil {
		return aiMessage, nil
	}
	fmt.Printf("⚠️ [Fallback] Groq lỗi: %v\n", err)

	return models.GeminiContent{}, fmt.Errorf("tất cả các provider đều thất bại")
}

>>>>>>> 7b74a62 (Add context window and conversation management)
func (a *Agent) GetHistory() []models.GeminiContent {
	a.mu.Lock()
	defer a.mu.Unlock()
	cp := make([]models.GeminiContent, len(a.conversation.ContextWindow.History))
	copy(cp, a.conversation.ContextWindow.History)
	return cp
}

func (a *Agent) logLoadedDocument(doc tools.LoadedDocument) {
	fmt.Printf("📎 [Sync] %s: %s (Size: %d chars)\n", strings.ToUpper(doc.DocType), doc.Name, len(doc.Content))
}
