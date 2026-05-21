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
	mu                sync.Mutex
	geminiProvider    *providers.GeminiProvider
	groqProvider      *providers.GroqProvider
	sambanovaProvider *providers.SambaNovaProvider
	history           []models.GeminiContent
	systemPrompt      string
	tools             []models.Parameters
	userInput         string
	handoffPlan       *RoutePlan
}

func NewAgent() *Agent {
	utils.LoadEnv()

	return &Agent{
		geminiProvider:    newGeminiProvider(),
		groqProvider:      newGroqProvider(),
		sambanovaProvider: newSambanovaProvider(),
		history:           []models.GeminiContent{},
		systemPrompt:      buildGroundedSystemPrompt(time.Now()),
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

func newGroqProvider() *providers.GroqProvider {
	return &providers.GroqProvider{
		APIKey: os.Getenv("GROQ_API_KEY"),
		Model:  os.Getenv("GROQ_MODEL"),
	}
}

func newSambanovaProvider() *providers.SambaNovaProvider {
	return &providers.SambaNovaProvider{
		APIKey: os.Getenv("SAMBANOVA_API_KEY"),
		Model:  os.Getenv("SAMBANOVA_MODEL"),
	}
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
	a.history = []models.GeminiContent{}
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
	defer a.mu.Unlock()
	a.userInput = userInput

	isNewConversation := len(a.history) == 0

	if isNewConversation {
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

	return a.runConversationLoopInternal()
}

func (a *Agent) handleSlashCommandInternal(input string) bool {
	parts := strings.Fields(input)
	cmd := strings.ToLower(parts[0])
	args := strings.Join(parts[1:], " ")

	var route RoutePlan
	switch cmd {
	case "/earnings":
		route = RoutePlan{Agent: "earnings-reviewer", Skills: []string{"earnings-analysis"}, Reason: "Slash command /earnings"}
	case "/market":
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
	BroadcastLog(fmt.Sprintf("Nạp cấu hình cho %s...", route.Agent), "process")
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
	a.history = append(a.history, models.GeminiContent{
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
		aiMessage, err := a.askProvidersInternal()
		if err != nil {
			return "", err
		}

		a.history = append(a.history, aiMessage)
		hasToolCall := a.handleToolCalls(aiMessage)

		if a.handoffPlan != nil {
			plan := *a.handoffPlan
			a.handoffPlan = nil
			fmt.Printf("\n🔀 [Orchestrator] Executing handoff to: %s\n", plan.Agent)
			a.executeBootstrapWithRouteInternal(plan)
			continue
		}

		if !hasToolCall {
			for i := len(aiMessage.Parts) - 1; i >= 0; i-- {
				if aiMessage.Parts[i].Text != "" {
					return aiMessage.Parts[i].Text, nil
				}
			}
			return "", nil
		}
	}
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

func (a *Agent) askProvidersInternal() (models.GeminiContent, error) {
	// 1. Gemini (Primary)
	aiMessage, err := a.geminiProvider.Call(a.systemPrompt, a.history, a.tools...)
	if err == nil {
		return aiMessage, nil
	}
	fmt.Printf("⚠️ [Fallback] Gemini lỗi: %v\n", err)

	// 2. SambaNova (Backup 1 - High Rate Limit Free)
	aiMessage, err = a.sambanovaProvider.Call(a.systemPrompt, a.history, a.tools...)
	if err == nil {
		return aiMessage, nil
	}
	fmt.Printf("⚠️ [Fallback] SambaNova lỗi: %v\n", err)

	// 3. Groq (Backup 2)
	aiMessage, err = a.groqProvider.Call(a.systemPrompt, a.history, a.tools...)
	if err != nil {
		if strings.Contains(err.Error(), "Rate limit reached") {
			fmt.Println("⏳ [Hệ thống] Chạm giới hạn Groq TPM. Đang chờ 15s để thử lại...")
			time.Sleep(15 * time.Second)
			aiMessage, err = a.groqProvider.Call(a.systemPrompt, a.history, a.tools...)
		}
	}
	if err == nil {
		return aiMessage, nil
	}
	fmt.Printf("⚠️ [Fallback] Groq lỗi: %v\n", err)

	return models.GeminiContent{}, fmt.Errorf("tất cả các provider đều thất bại")
}

func (a *Agent) GetHistory() []models.GeminiContent {
	a.mu.Lock()
	defer a.mu.Unlock()
	cp := make([]models.GeminiContent, len(a.history))
	copy(cp, a.history)
	return cp
}

func normalizeGeminiModel(model string) string {
	normalized := strings.TrimSpace(model)
	if normalized == "" {
		return "gemini-1.5-flash"
	}
	return normalized
}

func (a *Agent) logLoadedDocument(doc tools.LoadedDocument) {
	fmt.Printf("📎 [Sync] %s: %s (Size: %d chars)\n", strings.ToUpper(doc.DocType), doc.Name, len(doc.Content))
}
