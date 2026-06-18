package core

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
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
	mu                   sync.RWMutex
	pm                   *ProviderManager
	systemPrompt         string
	userInput            string
	handoffPlan          *RoutePlan
	conversation         *Conversation
	orchestrator         *Orchestrator
	dispatcher           *Dispatcher
	lastSystemPromptUpdate time.Time
}

func NewAgent() *Agent {
	utils.LoadEnv()

	a := &Agent{
		pm:                   NewProviderManager(),
		systemPrompt:         buildGroundedSystemPrompt(time.Now()),
		conversation:         NewConversation("default"),
		lastSystemPromptUpdate: time.Now(),
	}

	a.orchestrator = NewOrchestrator(a)
	a.dispatcher = NewDispatcher(a)

	// Bắt đầu background routine cập nhật system prompt
	go a.startSystemPromptUpdater()

	return a
}

// CheckTimeKnowledgeIssues kiểm tra và cảnh báo các vấn đề về thời gian
func (a *Agent) CheckTimeKnowledgeIssues(text string) string {
	now := time.Now()
	currentYear := now.Year()
	warnings := []string{}
	warningSet := make(map[string]bool) // Trùng lặp warnings

	// 1. Kiểm tra các năm cũ (2024 và trước đó khi đã có năm mới)
	if currentYear > 2024 {
		yearMatches := regexp.MustCompile(`\b2024\b`).FindAllString(text, -1)
		if len(yearMatches) > 0 {
			warningSet["- Dữ liệu năm 2024 có thể không còn mới nhất"] = true
		}

		// Kiểm tra các năm cũ khác
		oldYears := []int{2023, 2022, 2021}
		for _, year := range oldYears {
			yearMatches := regexp.MustCompile(`\b`+strconv.Itoa(year)+`\b`).FindAllString(text, -1)
			if len(yearMatches) > 0 {
				warningSet[fmt.Sprintf("- Tham chiếu đến năm %d có thể không cập nhật", year)] = true
			}
		}
	}

	// 2. Kiểm tra các biểu thức thời gian không hợp lý
	illogicalPatterns := []string{
		`3 năm trước`, `2 năm trước`, `1 năm trước`,
		`6 tháng trước`, `3 tháng trước`, `1 tháng trước`,
	}
	for _, pattern := range illogicalPatterns {
		if strings.Contains(text, pattern) {
			// Kiểm tra xem kết quả có phải là năm cũ không
			for _, year := range []int{2024, 2023, 2022} {
				if strings.Contains(text, strconv.Itoa(year)) {
					warningSet[fmt.Sprintf("- Biểu thức '%s' đề cập đến năm %d có thể không chính xác", pattern, year)] = true
					break
				}
			}
		}
	}

	// 3. Kiểm tra "gần đây" với các năm cũ
	recentPatterns := []string{"gần nhất", "gần đây", "hiện tại"}
	for _, pattern := range recentPatterns {
		if strings.Contains(text, pattern) {
			// Nếu chứa các pattern này nhưng lại đề cập đến năm 2024
			if strings.Contains(text, "2024") && currentYear > 2024 {
				warningSet[fmt.Sprintf("- Từ '%s' cần được xác nhận lại với dữ liệu mới nhất", pattern)] = true
			}
		}
	}

	// 4. Kiểm tra giờ giao dịch
	if strings.Contains(text, "giờ giao dịch") {
		hour := now.Hour()
		weekday := now.Weekday()
		if !(weekday >= time.Monday && weekday <= time.Friday && hour >= 9 && hour < 15) {
			warningSet["- Ngoài giờ giao dịch chính (9:00-15:00 từ thứ 2 đến thứ 6)"] = true
		}
	}

	// 5. Sử dụng time parser để validate các biểu thức thời gian cụ thể
	timeExpressions := regexp.MustCompile(`\d{1,2}[/-]\d{1,2}[/-]\d{4}|\d{4}`).FindAllString(text, -1)
	for _, expr := range timeExpressions {
		// Nếu là năm và là năm cũ
		if len(expr) == 4 {
			year, err := strconv.Atoi(expr)
			if err == nil && year < currentYear-1 {
				warningSet[fmt.Sprintf("- Dữ liệu năm %d có thể không còn mới", year)] = true
			}
		}
	}

	// Chuyển map thành slice để tránh trùng lặp
	for warning := range warningSet {
		warnings = append(warnings, warning)
	}

	if len(warnings) > 0 {
		result := "*⚠️ CẢNH BÁO VỀ THỜI GIAN:*\n"
		result += strings.Join(warnings, "\n")
		result += fmt.Sprintf("\n\nHiện tại là: %s (%d)\n", now.Format("02/01/2006"), currentYear)
		result += "Vui lòng kiểm tra với nguồn dữ liệu mới nhất.\n\n"
		return result
	}

	return ""
}

// EnsureFreshSystemPrompt đảm bảo system prompt luôn cập nhật nhất
// Nên được gọi trước mỗi lần gọi LLM
func (a *Agent) EnsureFreshSystemPrompt() {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Cập nhật nếu system prompt cũ hơn 1 giờ
	now := time.Now()
	if now.Sub(a.lastSystemPromptUpdate) >= time.Hour {
		a.systemPrompt = buildGroundedSystemPrompt(now)
		a.lastSystemPromptUpdate = now
	}
}

// startSystemPromptUpdater chạy routine background để cập nhật định kỳ
func (a *Agent) startSystemPromptUpdater() {
	ticker := time.NewTicker(30 * time.Minute) // Cập nhật mỗi 30 phút
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.EnsureFreshSystemPrompt()
		}
	}
}


func buildGroundedSystemPrompt(now time.Time) string {
	currentYear := now.Year()
	return utils.RenderPromptTemplate("enhanced_time_prompt.txt", map[string]string{
		"SYSTEM_TIME":        now.Format("02/01/2006 15:04:05"),
		"SYSTEM_WEEKDAY":     utils.TranslateWeekday(now.Weekday()),
		"CURRENT_YEAR":       fmt.Sprintf("%d", currentYear),
		"YEAR_MINUS_1":       fmt.Sprintf("%d", currentYear-1),
		"YEAR_MINUS_2":       fmt.Sprintf("%d", currentYear-2),
		"YEAR_MINUS_3":       fmt.Sprintf("%d", currentYear-3),
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
	// Đảm bảo system prompt luôn mới nhất
	a.EnsureFreshSystemPrompt()
	return a.orchestrator.ProcessMessage(ctx, userInput, atts)
}

// ProcessMessageStream processes a message and streams tokens via onChunk.
// Locking is managed internally by the orchestrator for minimal critical sections.
func (a *Agent) ProcessMessageStream(ctx context.Context, userInput string, atts []messaging.Attachment, onChunk func(string, bool)) error {
	// Đảm bảo system prompt luôn mới nhất
	a.EnsureFreshSystemPrompt()
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
