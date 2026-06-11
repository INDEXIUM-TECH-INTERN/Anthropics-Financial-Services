package core

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"gemini-cli/internal/models/messaging"
	"gemini-cli/internal/providers"
	"gemini-cli/internal/pubsub"
	"gemini-cli/internal/tools"
	"gemini-cli/internal/utils"
)

type Orchestrator struct {
	agent *Agent
}

func NewOrchestrator(a *Agent) *Orchestrator {
	return &Orchestrator{agent: a}
}

func (o *Orchestrator) ProcessMessage(userInput string, atts []messaging.Attachment) (string, error) {
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
				pubsub.BroadcastLog("Nhận diện ý định xã giao. Đang phản hồi nhanh...", "routing")
				o.agent.appendUserTextInternal(userInput, atts)
				o.agent.mu.Unlock()
				return o.runConversationLoopInternal()
			}

			pubsub.BroadcastLog("Khởi tạo cuộc hội thoại mới...", "process")
			o.agent.appendUserTextInternal(userInput, atts)
			o.agent.mu.Unlock() // Unlock before calling bootstrapContextInternal to avoid deadlock
			o.bootstrapContextInternal()
			return o.runConversationLoopInternal()
		}
	} else {
		o.agent.appendUserTextInternal(userInput, atts)
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
	if len(parts) == 0 {
		return false
	}
	cmd := strings.ToLower(parts[0])
	args := strings.Join(parts[1:], " ")

	var route RoutePlan
	var agent string
	var skills []string

	switch cmd {
	case "/pitch-deck", "/pitch":
		agent = "pitch-agent"
		skills = []string{"pitch-deck"}
	case "/datapack":
		agent = "pitch-agent"
		skills = []string{"datapack-builder"}
	case "/cim":
		agent = "pitch-agent"
		skills = []string{"cim-builder"}
	case "/teaser":
		agent = "pitch-agent"
		skills = []string{"teaser"}
	case "/buyer-list":
		agent = "pitch-agent"
		skills = []string{"buyer-list"}
	case "/precedents":
		agent = "pitch-agent"
		skills = []string{"precedent-transactions"}
	case "/briefing":
		agent = "meeting-prep-agent"
		skills = []string{"briefing-pack"}
	case "/bio":
		agent = "meeting-prep-agent"
		skills = []string{"biography-generator"}
	case "/profile":
		agent = "meeting-prep-agent"
		skills = []string{"company-profile"}
	case "/news":
		agent = "meeting-prep-agent"
		skills = []string{"news-digest"}
	case "/sector":
		agent = "market-researcher"
		skills = []string{"sector-overview"}
	case "/market":
		agent = "market-researcher"
		skills = []string{"sector-overview"}
	case "/competitors":
		agent = "market-researcher"
		skills = []string{"competitive-analysis"}
	case "/comps":
		agent = "market-researcher"
		skills = []string{"comps-analysis"}
	case "/ideas":
		agent = "market-researcher"
		skills = []string{"idea-generation"}
	case "/thesis":
		agent = "market-researcher"
		skills = []string{"thesis-tracker"}
	case "/catalyst":
		agent = "market-researcher"
		skills = []string{"catalyst-calendar"}
	case "/earnings":
		agent = "earnings-reviewer"
		skills = []string{"earnings-analysis"}
	case "/preview":
		agent = "earnings-reviewer"
		skills = []string{"earnings-preview"}
	case "/ic-memo":
		agent = "earnings-reviewer"
		skills = []string{"initiating-coverage"}
	case "/update-model":
		agent = "earnings-reviewer"
		skills = []string{"model-update"}
	case "/morning-note":
		agent = "earnings-reviewer"
		skills = []string{"morning-note"}
	case "/earnings-xlsx":
		agent = "earnings-reviewer"
		skills = []string{"xlsx-author"}
	case "/dcf":
		agent = "model-builder"
		skills = []string{"dcf-model"}
	case "/dcf-model":
		agent = "model-builder"
		skills = []string{"dcf-model"}
	case "/lbo":
		agent = "model-builder"
		skills = []string{"lbo-model"}
	case "/model-3s":
		agent = "model-builder"
		skills = []string{"3-statement-model"}
	case "/merger":
		agent = "model-builder"
		skills = []string{"merger-model"}
	case "/model-xlsx":
		agent = "model-builder"
		skills = []string{"xlsx-author"}
	case "/audit-xls":
		agent = "model-builder"
		skills = []string{"audit-xls"}
	case "/valuation-review":
		agent = "valuation-reviewer"
		skills = []string{"valuation-review"}
	case "/gp-reporting":
		agent = "valuation-reviewer"
		skills = []string{"gp-reporting"}
	case "/portfolio":
		agent = "valuation-reviewer"
		skills = []string{"lp-reporting"}
	case "/lp-reporting":
		agent = "valuation-reviewer"
		skills = []string{"lp-reporting"}
	case "/breaks":
		agent = "gl-reconciler"
		skills = []string{"break-detection"}
	case "/root-cause":
		agent = "gl-reconciler"
		skills = []string{"root-cause-analysis"}
	case "/sign-off":
		agent = "gl-reconciler"
		skills = []string{"sign-off-routing"}
	case "/accruals":
		agent = "month-end-closer"
		skills = []string{"accruals"}
	case "/roll-forwards":
		agent = "month-end-closer"
		skills = []string{"roll-forwards"}
	case "/variance":
		agent = "month-end-closer"
		skills = []string{"variance-commentary"}
	case "/help":
		fmt.Println("\n-- ANTHROPIC CLI SIMULATOR COMMANDS --")
		fmt.Println("/earnings <ticker> : Run Earnings Reviewer workflow")
		fmt.Println("/market <query>   : Run Market Researcher workflow")
		return false
	default:
		return false
	}

	if args == "" {
		switch cmd {
		case "/pitch-deck", "/pitch":
			args = "Tạo tài liệu Pitch Deck giới thiệu cơ hội đầu tư chuyên nghiệp."
		case "/datapack":
			args = "Xây dựng gói dữ liệu tài chính phục vụ phân tích đầu tư."
		case "/cim":
			args = "Thực hiện soạn thảo Bản thông tin ghi nhớ chi tiết (CIM - Confidential Information Memorandum)."
		case "/teaser":
			args = "Tạo bản tóm tắt cơ hội đầu tư dự án (Teaser)."
		case "/buyer-list":
			args = "Lập danh sách người mua hoặc đối tác tiềm năng phù hợp."
		case "/precedents":
			args = "Phân tích các giao dịch tiền lệ tương tự trong ngành."
		case "/briefing":
			args = "Tóm tắt tài liệu họp chi tiết cho ban lãnh đạo."
		case "/bio":
			args = "Tạo hồ sơ tiểu sử chi tiết của thành viên ban lãnh đạo."
		case "/profile":
			args = "Lập hồ sơ giới thiệu thông tin doanh nghiệp chi tiết."
		case "/news":
			args = "Tóm tắt tin tức thị trường và sự kiện quan trọng gần đây."
		case "/sector", "/market":
			args = "Thực hiện phân tích tổng quan ngành và xu hướng thị trường."
		case "/competitors":
			args = "Phân tích đối thủ cạnh tranh và vị thế của doanh nghiệp trong ngành."
		case "/comps":
			args = "Phân tích so sánh ngang hàng (peer comps) định giá các doanh nghiệp tương đồng."
		case "/ideas":
			args = "Đề xuất và đánh giá các ý tưởng đầu tư tiềm năng."
		case "/thesis":
			args = "Cập nhật và theo dõi các giả định/luận điểm đầu tư chính."
		case "/catalyst":
			args = "Cập nhật lịch sự kiện và các yếu tố xúc tác thị trường quan trọng."
		case "/earnings":
			args = "Đánh giá kết quả kinh doanh và báo cáo tài chính gần nhất."
		case "/preview":
			args = "Phân tích dự báo kết quả kinh doanh quý/năm sắp tới."
		case "/ic-memo":
			args = "Thực hiện báo cáo phân tích khởi đầu (Initiating Coverage Memo) cho doanh nghiệp."
		case "/update-model":
			args = "Thực hiện cập nhật mô hình tài chính với số liệu mới nhất."
		case "/morning-note":
			args = "Tạo bản tin phân tích buổi sáng (Morning Note) tóm tắt các điểm đáng chú ý."
		case "/earnings-xlsx":
			args = "Tạo báo cáo tài chính định dạng Excel (.xlsx) chuyên nghiệp."
		case "/dcf", "/dcf-model":
			args = "Thực hiện định giá theo phương pháp chiết khấu dòng tiền (DCF valuation)."
		case "/lbo":
			args = "Xây dựng mô hình mua lại có tài trợ nợ (LBO valuation model)."
		case "/model-3s":
			args = "Xây dựng mô hình tài chính 3 báo cáo (3-statement financial model) liên kết."
		case "/merger":
			args = "Phân tích và mô phỏng tác động của giao dịch sáp nhập (M&A Merger model)."
		case "/model-xlsx":
			args = "Xuất mô hình tài chính sang file Excel (.xlsx) với các công thức chuẩn xác."
		case "/audit-xls":
			args = "Kiểm tra lỗi, kiểm toán công thức và tính nhất quán của file Excel tài chính."
		case "/valuation-review":
			args = "Thực hiện kiểm tra và soát xét gói định giá doanh nghiệp."
		case "/gp-reporting":
			args = "Soạn thảo báo cáo định kỳ cho Quỹ đầu tư (GP reporting)."
		case "/portfolio", "/lp-reporting":
			args = "Soạn thảo báo cáo kết quả danh mục đầu tư gửi cho LP (LP reporting)."
		case "/breaks":
			args = "Thực hiện đối soát sổ cái và phát hiện các điểm sai lệch/bất thường."
		case "/root-cause":
			args = "Phân tích nguyên nhân gốc rễ của các sai lệch số liệu kế toán."
		case "/sign-off":
			args = "Thực hiện quy trình duyệt và luân chuyển ký duyệt báo cáo tài chính."
		case "/accruals":
			args = "Tính toán và ghi nhận các khoản chi phí dồn tích cuối kỳ."
		case "/roll-forwards":
			args = "Thực hiện đối chiếu số dư lũy kế đầu kỳ và cuối kỳ."
		case "/variance":
			args = "Phân tích và giải trình các biến động lớn giữa số thực tế và dự toán."
		}
	}

	pubsub.BroadcastLog(fmt.Sprintf("Kích hoạt lệnh %s...", cmd), "routing")
	route = RoutePlan{
		Agent:  agent,
		Skills: skills,
		Reason: fmt.Sprintf("Slash command %s", cmd),
	}

	o.agent.userInput = args
	o.agent.appendUserTextInternal(args, nil)
	o.executeBootstrapWithRouteInternal(route)
	return true
}

func (o *Orchestrator) bootstrapContextInternal() {
	route := o.agent.selectRoutePlan()
	fmt.Printf("🧭 [Router] Identified Agent: %s (Reason: %s)\n", route.Agent, route.Reason)
	o.executeBootstrapWithRouteInternal(route)
}

func (o *Orchestrator) executeBootstrapWithRouteInternal(route RoutePlan) {
	pubsub.BroadcastLog(fmt.Sprintf("Nạp cấu hình cho Agent: %s...", route.Agent), "routing")
	fmt.Printf("🧭 [Context] Orchestrator: Loading %s configuration...\n", route.Agent)
	contextParts := o.buildBootstrapContextInternal(route)
	bootstrapPayload := strings.Join(contextParts, "\n\n")
	o.agent.appendUserTextInternal(bootstrapPayload, nil)
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
		pubsub.BroadcastEvent(fmt.Sprintf("Đang nạp skill chuyên biệt: %s", skill), "skill_loaded", map[string]interface{}{
			"skill": skill,
		})
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
		pubsub.BroadcastLog("Phát hiện nhu cầu dữ liệu Real-time. Đang tìm kiếm...", "process")
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

// ProcessMessageStream xử lý chat với real streaming từ LLM provider.
// Thay vì split reply thành words (fake streaming), hàm này dùng GenerateStream
// để stream tokens thực tế từ provider.
// Lưu ý: Tool calls không hỗ trợ streaming — nếu AI cần gọi tool, streaming sẽ
// chuyển sang chế độ blocking cho đến khi tool xong, rồi stream final response.
func (o *Orchestrator) ProcessMessageStream(userInput string, atts []messaging.Attachment, onChunk func(string, bool)) error {
	// Phase 1: Bootstrap context (giống ProcessMessage nhưng không stream)
	o.agent.mu.Lock()
	o.agent.userInput = userInput
	isNewConversation := len(o.agent.conversation.ContextWindow.History) == 0

	if isNewConversation {
		if strings.HasPrefix(userInput, "/") {
			if o.handleSlashCommandInternal(userInput) {
				// handled
			}
		} else {
			if isCasualGreeting(userInput) {
				pubsub.BroadcastLog("Nhận diện ý định xã giao. Đang phản hồi nhanh...", "routing")
				o.agent.appendUserTextInternal(userInput, atts)
				o.agent.mu.Unlock()
				return o.streamFinalResponse(onChunk)
			}
			pubsub.BroadcastLog("Khởi tạo cuộc hội thoại mới...", "process")
			o.agent.appendUserTextInternal(userInput, atts)
			o.agent.mu.Unlock()
			o.bootstrapContextInternal()
			return o.streamFinalResponse(onChunk)
		}
	} else {
		o.agent.appendUserTextInternal(userInput, atts)
	}
	o.agent.mu.Unlock()

	return o.streamFinalResponse(onChunk)
}

// streamFinalResponse chạy ReAct loop nhưng với streaming cho LLM calls.
// Mỗi iteration: gọi GenerateStream → collect tokens → nếu có tool call thì execute (blocking)
// → lặp lại cho đến khi AI trả về text response không có tool call → stream tokens.
func (o *Orchestrator) streamFinalResponse(onChunk func(string, bool)) error {
	keepRecentMessages := getEnvInt("CONTEXT_KEEP_RECENT", 7)
	maxContextTokens := getEnvInt("CONTEXT_MAX_TOKENS", 92000)
	maxSummaryChars := getEnvInt("CONTEXT_MAX_SUMMARY_INPUT", 18000)

	for {
		// Kiểm tra context summarization
		o.agent.mu.Lock()
		cw := o.agent.conversation.ContextWindow
		needsSummary := cw.ShouldSummarize(maxContextTokens, keepRecentMessages)
		o.agent.mu.Unlock()

		if needsSummary {
			pubsub.BroadcastLog("Context window lớn, đang tóm tắt lịch sử cũ...", "process")
			_, err := cw.SummarizeOldest(o.agent.GetProvider(), keepRecentMessages, maxSummaryChars)
			if err != nil {
				fmt.Printf("⚠️ [Context] Tóm tắt thất bại: %v.\n", err)
			}
		}

		// Build messages
		o.agent.mu.Lock()
		systemPrompt := o.agent.systemPrompt
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

		req := messaging.Request{
			History: messages,
			Tools:   tools,
		}

		// Gọi LLM với streaming thực tế
		var fullText strings.Builder
		streamDone := make(chan error, 1)

		go func() {
			err := o.agent.GetProvider().GenerateStream(context.Background(), req, func(sc providers.StreamChunk) {
				if sc.Text != "" {
					fullText.WriteString(sc.Text)
					onChunk(sc.Text, false)
				}
				if sc.Done {
					onChunk("", true)
				}
			})
			streamDone <- err
		}()

		select {
		case err := <-streamDone:
			if err != nil {
				return err
			}
		case <-time.After(10 * time.Minute):
			return fmt.Errorf("streaming timeout")
		}

		// Append vào history
		finalText := fullText.String()
		o.agent.mu.Lock()
		o.agent.conversation.ContextWindow.History = append(o.agent.conversation.ContextWindow.History, messaging.Message{
			Role:    messaging.RoleAssistant,
			Content: finalText,
		})
		hasToolCall := o.agent.dispatcher.HandleToolCalls(o.agent.conversation.ContextWindow.History[len(o.agent.conversation.ContextWindow.History)-1])

		if o.agent.handoffPlan != nil {
			plan := *o.agent.handoffPlan
			o.agent.handoffPlan = nil
			o.executeBootstrapWithRouteInternal(plan)
			o.agent.mu.Unlock()
			continue
		}
		o.agent.mu.Unlock()

		if !hasToolCall {
			return nil
		}
		// Có tool call → loop tiếp (tool execution là blocking, response sẽ stream ở iteration sau)
	}
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
			pubsub.BroadcastLog("Context window lớn, đang tóm tắt lịch sử cũ...", "process")
			fmt.Printf("🧠 [Context] Đang tóm tắt %d tin nhắn cũ để tiết kiệm context...\n", len(cw.History)-keepRecentMessages)

			// Gọi tóm tắt (dùng provider hiện tại)
			_, err := cw.SummarizeOldest(o.agent.GetProvider(), keepRecentMessages, maxSummaryChars)
			if err != nil {
				fmt.Printf("⚠️ [Context] Tóm tắt thất bại: %v. Tiếp tục với context đầy đủ.\n", err)
				pubsub.BroadcastLog("Tóm tắt context thất bại, tiếp tục với lịch sử gốc.", "error")
			} else {
				pubsub.BroadcastLog("Đã tóm tắt thành công. Context đã được nén.", "success")
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

		aiMessage, err := o.agent.GetProvider().Generate(context.Background(), req)
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
