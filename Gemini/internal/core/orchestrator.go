package core

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"gemini-cli/internal/api"
	"gemini-cli/internal/models/messaging"
	"gemini-cli/internal/tools"
	"gemini-cli/internal/utils"
)

type Orchestrator struct {
	agent *Agent
}

func NewOrchestrator(a *Agent) *Orchestrator {
	return &Orchestrator{agent: a}
}

func (o *Orchestrator) ProcessMessage(userInput string) (string, error) {
	o.agent.mu.Lock()
	o.agent.userInput = userInput

	isNewConversation := len(o.agent.conversation.ContextWindow.History) == 0

	if isNewConversation {
		if strings.HasPrefix(userInput, "/") {
			if o.handleSlashCommandInternal(userInput) {
				// handled
			}
		} else {
			// KIỂM TRA NHANH: Nếu là câu chào hỏi hoặc câu ngắn xã giao -> Bỏ qua Routing nặng nề
			if isCasualGreeting(userInput) {
				api.BroadcastLog("Nhận diện ý định xã giao. Đang phản hồi nhanh...", "routing")
				o.agent.appendUserTextInternal(userInput)
				o.agent.mu.Unlock()
				return o.runConversationLoopInternal()
			}

			api.BroadcastLog("Khởi tạo cuộc hội thoại mới...", "process")
			o.agent.appendUserTextInternal(userInput)
			o.agent.mu.Unlock() // Unlock before calling bootstrapContextInternal to avoid deadlock
			o.bootstrapContextInternal()
			return o.runConversationLoopInternal()
		}
	} else {
		o.agent.appendUserTextInternal(userInput)
	}
	o.agent.mu.Unlock()

	return o.runConversationLoopInternal()
}

func isCasualGreeting(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	// Remove basic punctuation
	lower = strings.NewReplacer(".", "", "!", "", "?", "", ",", "").Replace(lower)
	
	// Convert Vietnamese accents to unsigned for robust matching
	lower = removeAccents(lower)

	greetings := []string{
		"hi", "hello", "xin chao", "chao ban", "chao", "hey", "alo", 
		"ten ban la gi", "ban la ai", "ai do", "who are you",
		"giup toi", "huong dan", "su dung", "test",
	}
	
	for _, g := range greetings {
		if lower == g || strings.HasPrefix(lower, g+" ") || strings.Contains(lower, "la ai") || strings.Contains(lower, "ten gi") {
			return true
		}
	}
	return len(lower) < 5 
}

func removeAccents(s string) string {
	accents := map[string]string{
		"a": "áàảãạăắằẳẵặâấầẩẫậ",
		"d": "đ",
		"e": "éèẻẽẹêếềểễệ",
		"i": "íìỉĩị",
		"o": "óòỏõọôốồổỗộơớờởỡợ",
		"u": "úùủũụưứừửữự",
		"y": "ýỳỷỹỵ",
	}
	for unaccented, accentedChars := range accents {
		for _, char := range accentedChars {
			s = strings.ReplaceAll(s, string(char), unaccented)
		}
	}
	return s
}

func (o *Orchestrator) handleSlashCommandInternal(input string) bool {
	parts := strings.Fields(input)
	cmd := strings.ToLower(parts[0])
	args := strings.Join(parts[1:], " ")

	var route RoutePlan
	switch cmd {
	case "/earnings":
		api.BroadcastLog("Kích hoạt lệnh /earnings...", "routing")
		route = RoutePlan{Agent: "earnings-reviewer", Skills: []string{"earnings-analysis"}, Reason: "Slash command /earnings"}
	case "/market":
		api.BroadcastLog("Kích hoạt lệnh /market...", "routing")
		route = RoutePlan{Agent: "market-researcher", Skills: []string{"sector-overview"}, Reason: "Slash command /market"}
	case "/help":
		fmt.Println("\n-- ANTHROPIC CLI SIMULATOR COMMANDS --")
		fmt.Println("/earnings <ticker> : Run Earnings Reviewer workflow")
		fmt.Println("/market <query>   : Run Market Researcher workflow")
		return false
	default:
		return false
	}

	o.agent.userInput = args
	o.agent.appendUserTextInternal(args)
	o.executeBootstrapWithRouteInternal(route)
	return true
}

func (o *Orchestrator) bootstrapContextInternal() {
	route := o.agent.selectRoutePlan()
	fmt.Printf("🧭 [Router] Identified Agent: %s (Reason: %s)\n", route.Agent, route.Reason)
	o.executeBootstrapWithRouteInternal(route)
}

func (o *Orchestrator) executeBootstrapWithRouteInternal(route RoutePlan) {
	api.BroadcastLog(fmt.Sprintf("Nạp cấu hình cho Agent: %s...", route.Agent), "routing")
	fmt.Printf("🧭 [Context] Orchestrator: Loading %s configuration...\n", route.Agent)
	contextParts := o.buildBootstrapContextInternal(route)
	bootstrapPayload := strings.Join(contextParts, "\n\n")
	o.agent.appendUserTextInternal(bootstrapPayload)
}

func (o *Orchestrator) buildBootstrapContextInternal(route RoutePlan) []string {
	agentDoc := tools.LoadDocumentWithMetadata("agent", route.Agent)
	o.logLoadedDocument(agentDoc)

	contextParts := []string{
		fmt.Sprintf("ANTHROPIC AGENT CONFIGURATION\nAgent: %s\nSkills: %s\nMode: Managed Agent (API)", route.Agent, strings.Join(route.Skills, ", ")),
		fmt.Sprintf("SYSTEM PROMPT (from agents/%s.md)\n%s", route.Agent, agentDoc.Content),
	}

	// Nạp tất cả các skills hợp lệ từ route plan
	for _, skill := range route.Skills {
		api.BroadcastLog(fmt.Sprintf("Đang nạp skill chuyên biệt: %s", skill), "process")
		skillDoc := tools.LoadDocumentWithMetadata("skill", route.Agent+"/"+skill)
		
		if strings.HasPrefix(skillDoc.Content, "Lỗi:") {
			fmt.Printf("⚠️ [Context] Không thể nạp skill %s: %s\n", skill, skillDoc.Content)
			continue
		}
		
		o.logLoadedDocument(skillDoc)

		content := skillDoc.Content
		// Giới hạn độ dài để tránh tràn context nếu skill quá lớn, nhưng vẫn đảm bảo lấy đủ ý
		if len(content) > 8000 {
			content = content[:8000] + "\n... [Nội dung bị cắt bớt để tối ưu hóa context]"
		}
		contextParts = append(contextParts, fmt.Sprintf("SKILL MARKDOWN (%s)\n%s", skill, content))
	}

	if tools.NeedsRealtimeData(o.agent.userInput) {
		api.BroadcastLog("Phát hiện nhu cầu dữ liệu Real-time. Đang tìm kiếm...", "process")
		queryPlan := tools.BuildMarketQueryPlan(o.agent.userInput)
		
		var googleResult, tavilyResult string
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			googleResult = tools.SearchGoogle(queryPlan.SearchQuery)
		}()
		go func() {
			defer wg.Done()
			tavilyResult = tools.SearchTavily(queryPlan.SearchQuery)
		}()
		wg.Wait()

		var combinedResults []string
		if googleResult != "" && !strings.HasPrefix(googleResult, "Lỗi") && !strings.HasPrefix(googleResult, "Error") {
			combinedResults = append(combinedResults, "--- TỪ GOOGLE SEARCH ---\n"+googleResult)
		}
		if tavilyResult != "" && !strings.HasPrefix(tavilyResult, "Lỗi") && !strings.HasPrefix(tavilyResult, "Error") {
			combinedResults = append(combinedResults, "--- TỪ TAVILY SEARCH ---\n"+tavilyResult)
		}

		if len(combinedResults) > 0 {
			contextParts = append(contextParts, fmt.Sprintf("REAL-TIME MARKET DATA\n%s", strings.Join(combinedResults, "\n\n")))
		}
	}

	contextParts = append(contextParts, utils.LoadPrompt("bootstrap_context_suffix.txt"))
	return contextParts
}

func (o *Orchestrator) runConversationLoopInternal() (string, error) {
	// === Cấu hình Summarization (có thể override bằng biến môi trường) ===
	keepRecentMessages := getEnvInt("CONTEXT_KEEP_RECENT", 7)
	maxContextTokens := getEnvInt("CONTEXT_MAX_TOKENS", 92000)
	maxSummaryChars := getEnvInt("CONTEXT_MAX_SUMMARY_INPUT", 18000)

	// Ghi chú: 92000 an toàn cho hầu hết model 128k. Với Gemini 1M+ có thể tăng lên 300k-500k.

	for {
		// === BƯỚC 1: KIỂM TRA VÀ TÓM TẮT NGỮ CẢNH NẾU CẦN (ngoài lock) ===
		o.agent.mu.Lock()
		cw := o.agent.conversation.ContextWindow
		needsSummary := cw.ShouldSummarize(maxContextTokens, keepRecentMessages)
		o.agent.mu.Unlock()

		if needsSummary {
			api.BroadcastLog("Context window lớn, đang tóm tắt lịch sử cũ...", "process")
			fmt.Printf("🧠 [Context] Đang tóm tắt %d tin nhắn cũ để tiết kiệm context...\n", len(cw.History)-keepRecentMessages)

			// Gọi tóm tắt (dùng provider hiện tại)
			_, err := cw.SummarizeOldest(o.agent.provider, keepRecentMessages, maxSummaryChars)
			if err != nil {
				fmt.Printf("⚠️ [Context] Tóm tắt thất bại: %v. Tiếp tục với context đầy đủ.\n", err)
				api.BroadcastLog("Tóm tắt context thất bại, tiếp tục với lịch sử gốc.", "error")
			} else {
				api.BroadcastLog("Đã tóm tắt thành công. Context đã được nén.", "success")
				fmt.Printf("✅ [Context] Đã cập nhật MemorySummary và nén lịch sử.\n")
			}
		}

		// === BƯỚC 2: Xây dựng messages gửi cho LLM (dùng phiên bản đã nén) ===
		o.agent.mu.Lock()
		systemPrompt := o.agent.systemPrompt
		// Dùng BuildLLMHistory thay vì copy toàn bộ
		condensedHistory := o.agent.conversation.ContextWindow.BuildLLMHistory(keepRecentMessages)
		tools := o.agent.dispatcher.GetTools()
		o.agent.mu.Unlock()

		var messages []messaging.Message
		if systemPrompt != "" {
			messages = append(messages, messaging.Message{
				Role:    messaging.RoleSystem,
				Content: systemPrompt,
			})
		}
		messages = append(messages, condensedHistory...)

		// Log kích thước context đang dùng (rất hữu ích)
		estTokens := utils.EstimateFullPrompt(systemPrompt, extractHistoryTexts(condensedHistory), "tools")
		fmt.Printf("📏 [Context] Gửi ~%d tokens (gửi %d messages cho LLM: summary + bootstrap + tin gần nhất)\n", estTokens, len(condensedHistory))

		req := messaging.Request{
			History: messages,
			Tools:   tools,
		}

		aiMessage, err := o.agent.provider.Generate(context.Background(), req)
		if err != nil {
			return "", err
		}

		o.agent.mu.Lock()
		// Luôn append vào FULL history (để UI và lịch sử đầy đủ)
		o.agent.conversation.ContextWindow.History = append(o.agent.conversation.ContextWindow.History, aiMessage)
		hasToolCall := o.agent.dispatcher.HandleToolCalls(aiMessage)

		if o.agent.handoffPlan != nil {
			plan := *o.agent.handoffPlan
			o.agent.handoffPlan = nil
			fmt.Printf("\n🔀 [Orchestrator] Executing handoff to: %s\n", plan.Agent)
			o.executeBootstrapWithRouteInternal(plan)
			o.agent.mu.Unlock()
			continue
		}
		o.agent.mu.Unlock()

		if !hasToolCall {
			return extractResponseText(aiMessage), nil
		}
	}
}

// extractHistoryTexts hỗ trợ EstimateFullPrompt
func extractHistoryTexts(msgs []messaging.Message) []string {
	texts := make([]string, len(msgs))
	for i, m := range msgs {
		texts[i] = m.Content
	}
	return texts
}

func (o *Orchestrator) logLoadedDocument(doc tools.LoadedDocument) {
	fmt.Printf("📎 [Sync] %s: %s (Size: %d chars)\n", strings.ToUpper(doc.DocType), doc.Name, len(doc.Content))
}

func extractResponseText(aiMessage messaging.Message) string {
	return aiMessage.Content
}

// getEnvInt đọc biến môi trường dạng int, fallback nếu không có hoặc lỗi
func getEnvInt(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	if i, err := strconv.Atoi(val); err == nil {
		return i
	}
	return fallback
}
