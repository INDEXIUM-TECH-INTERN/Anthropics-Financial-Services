package core

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gemini-cli/internal/models/messaging"
	"gemini-cli/internal/providers"
	"gemini-cli/internal/pubsub"
	"gemini-cli/internal/routing"
	"gemini-cli/internal/utils"
)

var (
	// Pre-compiled regexes for stripThinkingTags — avoids regexp.MustCompile on every call
	thinkingDetailsRe = regexp.MustCompile(`(?is)<details[^>]*class="thinking-details"[^>]*>.*?</details>`)
	thinkingContentRe = regexp.MustCompile(`(?is)<div[^>]*class="thinking-content"[^>]*>.*?</div>`)
)

type Orchestrator struct {
	agent *Agent
}

func NewOrchestrator(a *Agent) *Orchestrator {
	return &Orchestrator{agent: a}
}

func (o *Orchestrator) ProcessMessage(ctx context.Context, userInput string, atts []messaging.Attachment) (string, error) {
	o.agent.mu.Lock()
	o.agent.userInput = userInput

	isNewConversation := len(o.agent.conversation.ContextWindow.History) == 0

	if isNewConversation {
		if strings.HasPrefix(userInput, "/") {
			if HandleSlashCommand(userInput, o.agent) {
				// handled
			}
		} else {
			if routing.IsCasualGreeting(userInput) {
				pubsub.BroadcastLog("Nhận diện ý định xã giao. Đang phản hồi nhanh...", "routing")
				o.agent.appendUserTextInternal(userInput, atts)
				o.agent.mu.Unlock()
				return o.runConversationLoopInternal(ctx)
			}

			pubsub.BroadcastLog("Khởi tạo cuộc hội thoại mới...", "process")
			o.agent.appendUserTextInternal(userInput, atts)
			o.agent.mu.Unlock()
			BootstrapContext(o.agent)
			return o.runConversationLoopInternal(ctx)
		}
	} else {
		o.agent.appendUserTextInternal(userInput, atts)
	}
	o.agent.mu.Unlock()

	return o.runConversationLoopInternal(ctx)
}

// ProcessMessageStream xử lý chat với real streaming từ LLM provider.
// Thay vì split reply thành words (fake streaming), hàm này dùng GenerateStream
// để stream tokens thực tế từ provider.
// Lưu ý: Tool calls không hỗ trợ streaming — nếu AI cần gọi tool, streaming sẽ
// chuyển sang chế độ blocking cho đến khi tool xong, rồi stream final response.
func (o *Orchestrator) ProcessMessageStream(ctx context.Context, userInput string, atts []messaging.Attachment, onChunk func(string, bool)) error {
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
			if routing.IsCasualGreeting(userInput) {
				pubsub.BroadcastLog("Nhận diện ý định xã giao. Đang phản hồi nhanh...", "routing")
				o.agent.appendUserTextInternal(userInput, atts)
				o.agent.mu.Unlock()
				return o.streamFinalResponse(ctx, onChunk)
			}
			pubsub.BroadcastLog("Khởi tạo cuộc hội thoại mới...", "process")
			o.agent.appendUserTextInternal(userInput, atts)
			o.agent.mu.Unlock()
			BootstrapContext(o.agent)
			return o.streamFinalResponse(ctx, onChunk)
		}
	} else {
		o.agent.appendUserTextInternal(userInput, atts)
	}
	o.agent.mu.Unlock()

	return o.streamFinalResponse(ctx, onChunk)
}

// streamFinalResponse chạy ReAct loop nhưng với streaming cho LLM calls.
// Mỗi iteration: gọi GenerateStream → collect tokens → nếu có tool call thì execute (blocking)
// → lặp lại cho đến khi AI trả về text response không có tool call → stream tokens.
func (o *Orchestrator) streamFinalResponse(ctx context.Context, onChunk func(string, bool)) error {
	keepRecentMessages := getEnvInt("CONTEXT_KEEP_RECENT", 7)
	maxContextTokens := getEnvInt("CONTEXT_MAX_TOKENS", 92000)
	maxSummaryChars := getEnvInt("CONTEXT_MAX_SUMMARY_INPUT", 18000)
	maxIterations := getEnvInt("REACT_MAX_ITERATIONS", 20)

	for i := 0; i < maxIterations; i++ {
		// Kiểm tra context summarization (read lock)
		o.agent.mu.RLock()
		cw := o.agent.conversation.ContextWindow
		needsSummary := cw.ShouldSummarize(maxContextTokens, keepRecentMessages)
		o.agent.mu.RUnlock()

		if needsSummary {
			pubsub.BroadcastLog("Context window lớn, đang tóm tắt lịch sử cũ...", "process")
			o.agent.mu.RLock()
			_, err := cw.SummarizeOldest(o.agent.GetProvider(), keepRecentMessages, maxSummaryChars)
			o.agent.mu.RUnlock()
			if err != nil {
				fmt.Printf("⚠️ [Context] Tóm tắt thất bại: %v.\n", err)
			}
		}

		// Build messages (read lock)
		o.agent.mu.RLock()
		systemPrompt := o.agent.systemPrompt
		condensedHistory := o.agent.conversation.ContextWindow.BuildLLMHistory(keepRecentMessages)
		tools := o.agent.dispatcher.GetTools()
		o.agent.mu.RUnlock()

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

		// Gọi LLM với streaming thực tế (ngoài lock)
		var fullText strings.Builder
		var accumulatedToolCalls []messaging.ToolCall
		streamDone := make(chan error, 1)

		go func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("⚠️ [Stream] Recovered from panic: %v\n", r)
					streamDone <- fmt.Errorf("stream panic: %v", r)
				}
			}()
			err := o.agent.GetProvider().GenerateStream(ctx, req, func(sc providers.StreamChunk) {
				if !sc.Done && sc.Text != "" {
					fullText.WriteString(sc.Text)
				}
				if len(sc.ToolCalls) > 0 {
					accumulatedToolCalls = append(accumulatedToolCalls, sc.ToolCalls...)
				}
			})
			streamDone <- err
		}()

		select {
		case err := <-streamDone:
			if err != nil {
				return err
			}
		case <-time.After(time.Duration(getEnvInt("STREAM_TIMEOUT_SECONDS", 600)) * time.Second):
			return fmt.Errorf("streaming timeout")
		}

		// Append vào history (write lock)
		finalText := fullText.String()
		msg := messaging.Message{
			Role:      messaging.RoleAssistant,
			Content:   finalText,
			ToolCalls: accumulatedToolCalls,
		}
		o.agent.mu.Lock()
		o.agent.conversation.ContextWindow.History = append(o.agent.conversation.ContextWindow.History, msg)
		o.agent.mu.Unlock()

		hasToolCall := o.agent.dispatcher.HandleToolCalls(msg)

		// === BƯỚC 5: KIỂM TRA VÀ CẢNH BÁO VẤN ĐỀ THỜI GIAN ===
		timeWarning := o.agent.CheckTimeKnowledgeIssues(finalText)
		if timeWarning != "" {
			fmt.Printf("⚠️ [Time] Phát hiện vấn đề thời gian trong response:\n%s", timeWarning)
			// Thêm cảnh báo vào finalText
			finalText += "\n\n" + timeWarning
			// Cập nhật lại message
			msg.Content = finalText
			o.agent.mu.Lock()
			o.agent.conversation.ContextWindow.History = append(o.agent.conversation.ContextWindow.History, msg)
			o.agent.mu.Unlock()
		}

		// Send only the final response to client (skip thinking/tool-call preamble)
		if !hasToolCall && finalText != "" {
			// Strip any leaked thinking HTML blocks that may have been prepended
			cleaned := stripThinkingTags(finalText)
			cleaned = strings.TrimSpace(cleaned)
			if cleaned != "" {
				runes := []rune(cleaned)
				chunkSize := 3
				for i := 0; i < len(runes); i += chunkSize {
					end := i + chunkSize
					if end > len(runes) {
						end = len(runes)
					}
					onChunk(string(runes[i:end]), false)
				}
			}
		}

		o.agent.mu.Lock()
		if o.agent.handoffPlan != nil {
			plan := *o.agent.handoffPlan
			o.agent.handoffPlan = nil
			ExecuteBootstrapWithRoute(o.agent, plan)
			o.agent.mu.Unlock()
			continue
		}
		o.agent.mu.Unlock()

		if !hasToolCall {
			onChunk("", true)
			return nil
		}
	}

	return fmt.Errorf("exceeded maximum ReAct iterations (%d); possible infinite tool-call loop", maxIterations)
}

func (o *Orchestrator) runConversationLoopInternal(ctx context.Context) (string, error) {
	keepRecentMessages := getEnvInt("CONTEXT_KEEP_RECENT", 7)
	maxContextTokens := getEnvInt("CONTEXT_MAX_TOKENS", 92000)
	maxSummaryChars := getEnvInt("CONTEXT_MAX_SUMMARY_INPUT", 18000)
	maxIterations := getEnvInt("REACT_MAX_ITERATIONS", 20)

	for i := 0; i < maxIterations; i++ {
		// === BƯỚC 1: KIỂM TRA VÀ TÓM TẮT NGỮ CẢNH NẾU CẦN (read lock) ===
		o.agent.mu.RLock()
		cw := o.agent.conversation.ContextWindow
		needsSummary := cw.ShouldSummarize(maxContextTokens, keepRecentMessages)
		o.agent.mu.RUnlock()

		if needsSummary {
			pubsub.BroadcastLog("Context window lớn, đang tóm tắt lịch sử cũ...", "process")
			fmt.Printf("🧠 [Context] Đang tóm tắt tin nhắn cũ để tiết kiệm context...\n")

			// SummarizeOldest reads history — hold read lock during the call
			o.agent.mu.RLock()
			_, err := cw.SummarizeOldest(o.agent.GetProvider(), keepRecentMessages, maxSummaryChars)
			o.agent.mu.RUnlock()

			if err != nil {
				fmt.Printf("⚠️ [Context] Tóm tắt thất bại: %v. Tiếp tục với context đầy đủ.\n", err)
				pubsub.BroadcastLog("Tóm tắt context thất bại, tiếp tục với lịch sử gốc.", "error")
			} else {
				pubsub.BroadcastLog("Đã tóm tắt thành công. Context đã được nén.", "success")
				fmt.Printf("✅ [Context] Đã cập nhật MemorySummary và nén lịch sử.\n")
			}
		}

		// === BƯỚC 2: Xây dựng messages gửi cho LLM (read lock) ===
		o.agent.mu.RLock()
		systemPrompt := o.agent.systemPrompt
		condensedHistory := o.agent.conversation.ContextWindow.BuildLLMHistory(keepRecentMessages)
		tools := o.agent.dispatcher.GetTools()
		o.agent.mu.RUnlock()

		var messages []messaging.Message
		if systemPrompt != "" {
			messages = append(messages, messaging.Message{
				Role:    messaging.RoleSystem,
				Content: systemPrompt,
			})
		}
		messages = append(messages, condensedHistory...)

		estTokens := utils.EstimateFullPrompt(systemPrompt, extractHistoryTexts(condensedHistory), "tools")
		fmt.Printf("📏 [Context] Gửi ~%d tokens (%d messages)\n", estTokens, len(condensedHistory))

		req := messaging.Request{
			History: messages,
			Tools:   tools,
		}

		// === BƯỚC 3: Gọi LLM (ngoài lock) ===
		aiMessage, err := o.agent.GetProvider().Generate(ctx, req)
		if err != nil {
			return "", err
		}

		// === BƯỚC 4: Append response (write lock), then handle tool calls (outside lock) ===
		o.agent.mu.Lock()
		o.agent.conversation.ContextWindow.History = append(o.agent.conversation.ContextWindow.History, aiMessage)
		o.agent.mu.Unlock()

		hasToolCall := o.agent.dispatcher.HandleToolCalls(aiMessage)

		// === BƯỚC 5: KIỂM TRA VÀ CẢNH BÁO VẤN ĐỀ THỜI GIAN ===
		timeWarning := o.agent.CheckTimeKnowledgeIssues(aiMessage.Content)
		if timeWarning != "" {
			fmt.Printf("⚠️ [Time] Phát hiện vấn đề thời gian trong response:\n%s", timeWarning)
			// Thêm cảnh báo vào response
			aiMessage.Content += "\n\n" + timeWarning
			o.agent.mu.Lock()
			o.agent.conversation.ContextWindow.History = append(o.agent.conversation.ContextWindow.History, aiMessage)
			o.agent.mu.Unlock()
		}

		o.agent.mu.Lock()
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

	return "", fmt.Errorf("exceeded maximum ReAct iterations (%d); possible infinite tool-call loop", maxIterations)
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
	return strings.TrimSpace(stripThinkingTags(aiMessage.Content))
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

// stripThinkingTags removes HTML thinking/reasoning blocks that leak into the final response.
func stripThinkingTags(text string) string {
	text = thinkingDetailsRe.ReplaceAllString(text, "")
	text = thinkingContentRe.ReplaceAllString(text, "")
	return text
}
