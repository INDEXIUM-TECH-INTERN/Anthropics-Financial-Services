package core

import (
	"fmt"
	"strings"

	"gemini-cli/internal/pubsub"
)

// slashCommand defines the routing target for a slash command.
type slashCommand struct {
	agent       string
	skills      []string
	defaultArgs string
}

// slashCommands maps command names to their routing configuration.
var slashCommands = map[string]slashCommand{
	"/pitch-deck":  {"pitch-agent", []string{"pitch-deck"}, "Tạo tài liệu Pitch Deck giới thiệu cơ hội đầu tư chuyên nghiệp."},
	"/pitch":       {"pitch-agent", []string{"pitch-deck"}, "Tạo tài liệu Pitch Deck giới thiệu cơ hội đầu tư chuyên nghiệp."},
	"/datapack":    {"pitch-agent", []string{"datapack-builder"}, "Xây dựng gói dữ liệu tài chính phục vụ phân tích đầu tư."},
	"/cim":         {"pitch-agent", []string{"cim-builder"}, "Thực hiện soạn thảo Bản thông tin ghi nhớ chi tiết (CIM - Confidential Information Memorandum)."},
	"/teaser":      {"pitch-agent", []string{"teaser"}, "Tạo bản tóm tắt cơ hội đầu tư dự án (Teaser)."},
	"/buyer-list":  {"pitch-agent", []string{"buyer-list"}, "Lập danh sách người mua hoặc đối tác tiềm năng phù hợp."},
	"/precedents":  {"pitch-agent", []string{"precedent-transactions"}, "Phân tích các giao dịch tiền lệ tương tự trong ngành."},
	"/briefing":    {"meeting-prep-agent", []string{"briefing-pack"}, "Tóm tắt tài liệu họp chi tiết cho ban lãnh đạo."},
	"/bio":         {"meeting-prep-agent", []string{"biography-generator"}, "Tạo hồ sơ tiểu sử chi tiết của thành viên ban lãnh đạo."},
	"/profile":     {"meeting-prep-agent", []string{"company-profile"}, "Lập hồ sơ giới thiệu thông tin doanh nghiệp chi tiết."},
	"/news":        {"meeting-prep-agent", []string{"news-digest"}, "Tóm tắt tin tức thị trường và sự kiện quan trọng gần đây."},
	"/sector":      {"market-researcher", []string{"sector-overview"}, "Thực hiện phân tích tổng quan ngành và xu hướng thị trường."},
	"/market":      {"market-researcher", []string{"sector-overview"}, "Thực hiện phân tích tổng quan ngành và xu hướng thị trường."},
	"/competitors": {"market-researcher", []string{"competitive-analysis"}, "Phân tích đối thủ cạnh tranh và vị thế của doanh nghiệp trong ngành."},
	"/comps":       {"market-researcher", []string{"comps-analysis"}, "Phân tích so sánh ngang hàng (peer comps) định giá các doanh nghiệp tương đồng."},
	"/ideas":       {"market-researcher", []string{"idea-generation"}, "Đề xuất và đánh giá các ý tưởng đầu tư tiềm năng."},
	"/thesis":      {"market-researcher", []string{"thesis-tracker"}, "Cập nhật và theo dõi các giả định/luận điểm đầu tư chính."},
	"/catalyst":    {"market-researcher", []string{"catalyst-calendar"}, "Cập nhật lịch sự kiện và các yếu tố xúc tác thị trường quan trọng."},
	"/earnings":    {"earnings-reviewer", []string{"earnings-analysis"}, "Đánh giá kết quả kinh doanh và báo cáo tài chính gần nhất."},
	"/preview":     {"earnings-reviewer", []string{"earnings-preview"}, "Phân tích dự báo kết quả kinh doanh quý/năm sắp tới."},
	"/ic-memo":     {"earnings-reviewer", []string{"initiating-coverage"}, "Thực hiện báo cáo phân tích khởi đầu (Initiating Coverage Memo) cho doanh nghiệp."},
	"/update-model": {"earnings-reviewer", []string{"model-update"}, "Thực hiện cập nhật mô hình tài chính với số liệu mới nhất."},
	"/morning-note": {"earnings-reviewer", []string{"morning-note"}, "Tạo bản tin phân tích buổi sáng (Morning Note) tóm tắt các điểm đáng chú ý."},
	"/earnings-xlsx": {"earnings-reviewer", []string{"xlsx-author"}, "Tạo báo cáo tài chính định dạng Excel (.xlsx) chuyên nghiệp."},
	"/dcf":         {"model-builder", []string{"dcf-model"}, "Thực hiện định giá theo phương pháp chiết khấu dòng tiền (DCF valuation)."},
	"/dcf-model":   {"model-builder", []string{"dcf-model"}, "Thực hiện định giá theo phương pháp chiết khấu dòng tiền (DCF valuation)."},
	"/lbo":         {"model-builder", []string{"lbo-model"}, "Xây dựng mô hình mua lại có tài trợ nợ (LBO valuation model)."},
	"/model-3s":    {"model-builder", []string{"3-statement-model"}, "Xây dựng mô hình tài chính 3 báo cáo (3-statement financial model) liên kết."},
	"/merger":      {"model-builder", []string{"merger-model"}, "Phân tích và mô phỏng tác động của giao dịch sáp nhập (M&A Merger model)."},
	"/model-xlsx":  {"model-builder", []string{"xlsx-author"}, "Xuất mô hình tài chính sang file Excel (.xlsx) với các công thức chuẩn xác."},
	"/audit-xls":   {"model-builder", []string{"audit-xls"}, "Kiểm tra lỗi, kiểm toán công thức và tính nhất quán của file Excel tài chính."},
	"/valuation-review": {"valuation-reviewer", []string{"valuation-review"}, "Thực hiện kiểm tra và soát xét gói định giá doanh nghiệp."},
	"/gp-reporting":     {"valuation-reviewer", []string{"gp-reporting"}, "Soạn thảo báo cáo định kỳ cho Quỹ đầu tư (GP reporting)."},
	"/portfolio":       {"valuation-reviewer", []string{"lp-reporting"}, "Soạn thảo báo cáo kết quả danh mục đầu tư gửi cho LP (LP reporting)."},
	"/lp-reporting":     {"valuation-reviewer", []string{"lp-reporting"}, "Soạn thảo báo cáo kết quả danh mục đầu tư gửi cho LP (LP reporting)."},
	"/breaks":       {"gl-reconciler", []string{"break-detection"}, "Thực hiện đối soát sổ cái và phát hiện các điểm sai lệch/bất thường."},
	"/root-cause":   {"gl-reconciler", []string{"root-cause-analysis"}, "Phân tích nguyên nhân gốc rễ của các sai lệch số liệu kế toán."},
	"/sign-off":     {"gl-reconciler", []string{"sign-off-routing"}, "Thực hiện quy trình duyệt và luân chuyển ký duyệt báo cáo tài chính."},
	"/accruals":     {"month-end-closer", []string{"accruals"}, "Tính toán và ghi nhận các khoản chi phí dồn tích cuối kỳ."},
	"/roll-forwards": {"month-end-closer", []string{"roll-forwards"}, "Thực hiện đối chiếu số dư lũy kế đầu kỳ và cuối kỳ."},
	"/variance":     {"month-end-closer", []string{"variance-commentary"}, "Phân tích và giải trình các biến động lớn giữa số thực tế và dự toán."},
}

// HandleSlashCommand processes a slash command input. It returns true if the
// input was recognized as a slash command and routed; false otherwise.
// The agent's internal state (userInput, conversation history, bootstrap) is
// updated as a side effect.
func HandleSlashCommand(input string, agent *Agent) bool {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return false
	}
	cmd := strings.ToLower(parts[0])
	args := strings.Join(parts[1:], " ")

	sc, ok := slashCommands[cmd]
	if !ok {
		return false
	}

	if args == "" {
		args = sc.defaultArgs
	}

	pubsub.BroadcastLog(fmt.Sprintf("Kích hoạt lệnh %s...", cmd), "routing")
	route := RoutePlan{
		Agent:  sc.agent,
		Skills: sc.skills,
		Reason: fmt.Sprintf("Slash command %s", cmd),
	}

	agent.userInput = args
	agent.appendUserTextInternal(args, nil)
	ExecuteBootstrapWithRoute(agent, route)
	return true
}

// isCasualGreeting returns true when the input looks like a short social
// greeting (e.g. "hi", "hello", "xin chao").
func isCasualGreeting(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	lower = strings.NewReplacer(".", "", "!", "", "?", "", ",", "").Replace(lower)
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

// removeAccents converts Vietnamese accented characters to their plain ASCII
// equivalents so that greeting detection works regardless of accent marks.
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
