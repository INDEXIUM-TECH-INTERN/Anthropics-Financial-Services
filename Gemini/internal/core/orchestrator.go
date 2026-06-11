package core

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gemini-cli/internal/models/messaging"
	"gemini-cli/internal/providers"
	"gemini-cli/internal/pubsub"
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
			if HandleSlashCommand(userInput, o.agent) {
				// handled
			}
		} else {
			if isCasualGreeting(userInput) {
				pubsub.BroadcastLog("Nhận diện ý định xã giao. Đang phản hồi nhanh...", "routing")
				o.agent.appendUserTextInternal(userInput, atts)
				o.agent.mu.Unlock()
				return o.runConversationLoopInternal()
			}

			pubsub.BroadcastLog("Khởi tạo cuộc hội thoại mới...", "process")
			o.agent.appendUserTextInternal(userInput, atts)
			o.agent.mu.Unlock()
			BootstrapContext(o.agent)
			return o.runConversationLoopInternal()
		}
	} else {
		o.agent.appendUserTextInternal(userInput, atts)
	}
	o.agent.mu.Unlock()

	return o.runConversationLoopInternal()
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
			if HandleSlashCommand(userInput, o.agent) {
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
			BootstrapContext(o.agent)
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
			ExecuteBootstrapWithRoute(o.agent, plan)
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
			ExecuteBootstrapWithRoute(o.agent, plan)
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
