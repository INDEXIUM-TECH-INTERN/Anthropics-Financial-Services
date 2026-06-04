package core

import (
	"fmt"
	"strings"

	"gemini-cli/internal/models/messaging"
	"gemini-cli/internal/utils"
)

type ContextWindow struct {
	History       []messaging.Message
	MemorySummary string
	// lastSummarizedIdx giúp biết đã tóm tắt đến đâu (để tối ưu sau này)
	lastSummarizedIdx int
}

func NewContextWindow() *ContextWindow {
	return &ContextWindow{
		History:           []messaging.Message{},
		MemorySummary:     "",
		lastSummarizedIdx: 0,
	}
}

func (cw *ContextWindow) AddMessage(msg messaging.Message) {
	cw.History = append(cw.History, msg)
}

// Reset xóa toàn bộ lịch sử và tóm tắt
func (cw *ContextWindow) Reset() {
	cw.History = []messaging.Message{}
	cw.MemorySummary = ""
	cw.lastSummarizedIdx = 0
}

// GetFullHistory trả về bản sao lịch sử đầy đủ (dùng cho UI, log, reset)
func (cw *ContextWindow) GetFullHistory() []messaging.Message {
	cp := make([]messaging.Message, len(cw.History))
	copy(cp, cw.History)
	return cp
}

// BuildLLMHistory xây dựng danh sách message sẽ thực sự gửi cho LLM.
// Chiến lược (phiên bản cải tiến):
// - Luôn giữ System prompt riêng (thêm ở orchestrator).
// - Nếu có MemorySummary → chèn tóm tắt ở đầu.
// - Luôn giữ 1-2 tin nhắn đầu (user gốc + bootstrap agent/skill) vì chúng chứa hướng dẫn quan trọng.
// - Giữ thêm N tin nhắn gần nhất.
// - Phần giữa được thay thế bằng MemorySummary.
func (cw *ContextWindow) BuildLLMHistory(keepRecent int) []messaging.Message {
	if len(cw.History) == 0 {
		return []messaging.Message{}
	}

	result := []messaging.Message{}

	// 1. Thêm tóm tắt nếu có
	if cw.MemorySummary != "" {
		summaryMsg := messaging.Message{
			Role:    messaging.RoleUser,
			Content: "=== TÓM TẮT NGỮ CẢNH TRƯỚC ĐÂY (QUAN TRỌNG - ĐỌC KỸ) ===\n" + cw.MemorySummary + "\n=== KẾT THÚC TÓM TẮT ===",
		}
		result = append(result, summaryMsg)
	}

	// 2. Luôn giữ tin nhắn đầu tiên (user query gốc) và tin nhắn bootstrap (thường là tin thứ 2)
	// Bootstrap chứa toàn bộ agent + skill instructions, rất quan trọng.
	protected := 2
	if len(cw.History) < protected {
		protected = len(cw.History)
	}

	for i := 0; i < protected; i++ {
		result = append(result, cw.History[i])
	}

	// 3. Thêm tin nhắn gần đây (tránh trùng với protected)
	startIdx := 0
	if len(cw.History) > keepRecent {
		startIdx = len(cw.History) - keepRecent
	}
	if startIdx < protected {
		startIdx = protected
	}

	for i := startIdx; i < len(cw.History); i++ {
		result = append(result, cw.History[i])
	}

	return result
}

// EstimateCurrentTokens ước tính nhanh token của toàn bộ history hiện tại (full)
func (cw *ContextWindow) EstimateCurrentTokens() int {
	total := 0
	for _, msg := range cw.History {
		total += utils.EstimateTokens(msg.Content)
		// Ước tính thô cho tool calls/responses
		for _, tc := range msg.ToolCalls {
			total += 10 + utils.EstimateTokens(tc.Name)
		}
		for _, tr := range msg.ToolResponses {
			total += 8 + utils.EstimateTokens(tr.Content)
		}
	}
	return total
}

// ShouldSummarize kiểm tra xem có nên tóm tắt không
func (cw *ContextWindow) ShouldSummarize(maxTokens int, keepRecent int) bool {
	if len(cw.History) <= keepRecent+2 {
		return false // Quá ít lịch sử thì chưa cần
	}
	est := cw.EstimateCurrentTokens()
	return est > maxTokens
}

// UpdateSummary cập nhật MemorySummary (gọi sau khi tóm tắt thành công)
func (cw *ContextWindow) UpdateSummary(newSummary string, summarizedUpToIdx int) {
	cw.MemorySummary = strings.TrimSpace(newSummary)
	if summarizedUpToIdx > cw.lastSummarizedIdx {
		cw.lastSummarizedIdx = summarizedUpToIdx
	}
}

// Note: Chúng ta KHÔNG trim History thật sự nữa.
// Lý do: Giữ lịch sử đầy đủ cho UI hiển thị và "New Chat".
// Tóm tắt chỉ ảnh hưởng đến những gì gửi cho LLM (qua BuildLLMHistory + MemorySummary).
// Nếu sau này muốn nén storage, có thể thêm logic lưu full history ra file riêng.

// SummarizeOldest sử dụng provider để tóm tắt phần lịch sử cũ.
// Đây là hàm "thực sự" thực hiện tóm tắt.
func (cw *ContextWindow) SummarizeOldest(
	provider interface {
		GenerateText(systemPrompt, userPrompt string) (string, error)
	},
	keepRecent int,
	maxSummaryInputChars int,
) (string, error) {

	if len(cw.History) <= keepRecent {
		return "", nil
	}

	// Lấy phần cần tóm tắt (từ đầu đến trước keepRecent)
	endIdx := len(cw.History) - keepRecent
	if endIdx <= 0 {
		return "", nil
	}

	oldMessages := cw.History[:endIdx]

	// Chuyển thành text để tóm tắt
	var sb strings.Builder
	sb.WriteString("Đây là lịch sử hội thoại cũ cần tóm tắt:\n\n")
	for _, msg := range oldMessages {
		roleLabel := string(msg.Role)
		if roleLabel == "assistant" {
			roleLabel = "Agent"
		} else if roleLabel == "tool" {
			roleLabel = "Tool Result"
		} else if roleLabel == "user" {
			roleLabel = "User"
		}
		sb.WriteString(fmt.Sprintf("[%s] %s\n", roleLabel, utils.TruncateTextForSummary(msg.Content, 1200)))

		// Thêm thông tin tool call/response ngắn gọn
		for _, tc := range msg.ToolCalls {
			sb.WriteString(fmt.Sprintf("  → Gọi tool: %s\n", tc.Name))
		}
		for _, tr := range msg.ToolResponses {
			sb.WriteString(fmt.Sprintf("  ← Kết quả %s: %s\n", tr.Name, utils.TruncateTextForSummary(tr.Content, 600)))
		}
		sb.WriteString("\n")
	}

	historyText := sb.String()
	if len([]rune(historyText)) > maxSummaryInputChars {
		historyText = utils.TruncateTextForSummary(historyText, maxSummaryInputChars)
	}

	// Load prompt tóm tắt
	summaryPrompt := utils.RenderPromptTemplate("context_summarizer.txt", map[string]string{
		"HISTORY_TEXT": historyText,
	})

	// Gọi LLM để tóm tắt (dùng GenerateText để nhẹ)
	summary, err := provider.GenerateText(
		"Bạn là trợ lý tóm tắt ngữ cảnh chính xác và ngắn gọn.",
		summaryPrompt,
	)
	if err != nil {
		return "", fmt.Errorf("lỗi khi gọi LLM tóm tắt: %w", err)
	}

	// Cập nhật summary (History đầy đủ vẫn được giữ để UI hiển thị lịch sử gốc)
	cw.UpdateSummary(summary, endIdx)

	return summary, nil
}
