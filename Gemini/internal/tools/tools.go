package tools

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"gemini-cli/internal/utils"
)

type MarketQueryPlan struct {
	SearchQuery string
	TimeNote    string
}

type LoadedDocument struct {
	DocType     string
	Name        string
	SourceURL   string
	Content     string
	ContentSize int
}

// --- Các hàm Tool (MCP-style) ---

func SearchGoogle(query string) string {
	apiKey := os.Getenv("SERPAPI_KEY")
	if apiKey == "" {
		return "Lỗi: Chưa cấu hình SERPAPI_KEY."
	}
	fmt.Printf("🌐 [Tool] Đang tìm kiếm: %s...\n", query)
	searchURL := fmt.Sprintf(
		"https://serpapi.com/search.json?engine=google&hl=vi&gl=vn&q=%s&api_key=%s",
		url.QueryEscape(query),
		url.QueryEscape(apiKey),
	)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(searchURL)
	if err != nil {
		return "Lỗi kết nối tìm kiếm."
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	// Giải mã JSON để lọc thông tin quan trọng
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return "Lỗi giải mã kết quả tìm kiếm."
	}

	var sections []string
	if metadata, ok := data["search_metadata"].(map[string]interface{}); ok {
		if processedAt := strings.TrimSpace(fmt.Sprintf("%v", metadata["processed_at"])); processedAt != "" && processedAt != "<nil>" {
			sections = append(sections, "Search Timestamp:\nprocessed_at: "+processedAt)
		}
	}

	// Lấy Answer Box (Nếu có - thường chứa giá cổ phiếu trực tiếp)
	if ab, ok := data["answer_box"].(map[string]interface{}); ok {
		var lines []string
		for _, key := range []string{"title", "stock", "price", "currency", "exchange", "snippet"} {
			value := strings.TrimSpace(fmt.Sprintf("%v", ab[key]))
			if value == "" || value == "<nil>" {
				continue
			}
			lines = append(lines, fmt.Sprintf("%s: %s", key, value))
		}
		if len(lines) > 0 {
			sections = append(sections, "Answer Box:\n"+strings.Join(lines, "\n"))
		}
	}

	if stockInfo, ok := data["stock_information"].(map[string]interface{}); ok {
		var lines []string
		for _, key := range []string{"stock", "exchange", "price", "currency", "price_movement", "market_cap"} {
			value := strings.TrimSpace(fmt.Sprintf("%v", stockInfo[key]))
			if value == "" || value == "<nil>" {
				continue
			}
			lines = append(lines, fmt.Sprintf("%s: %s", key, value))
		}
		if len(lines) > 0 {
			sections = append(sections, "Stock Information:\n"+strings.Join(lines, "\n"))
		}
	}

	// Lấy các kết quả hữu cơ đầu tiên
	if results, ok := data["organic_results"].([]interface{}); ok {
		var lines []string
		for i, res := range results {
			if i >= 3 { // Chỉ lấy 3 kết quả đầu
				break
			}
			r, ok := res.(map[string]interface{})
			if !ok {
				continue
			}
			title := strings.TrimSpace(fmt.Sprintf("%v", r["title"]))
			if title == "" || title == "<nil>" {
				continue
			}
			snippet := strings.TrimSpace(fmt.Sprintf("%v", r["snippet"]))
			if len(snippet) > 200 {
				snippet = snippet[:200] + "..."
			}
			link := strings.TrimSpace(fmt.Sprintf("%v", r["link"]))
			date := strings.TrimSpace(fmt.Sprintf("%v", r["date"]))

			line := fmt.Sprintf("- %s", title)
			if date != "" && date != "<nil>" {
				line += fmt.Sprintf(" [%s]", date)
			}
			if snippet != "" && snippet != "<nil>" {
				line += fmt.Sprintf(": %s", snippet)
			}
			if link != "" && link != "<nil>" {
				line += fmt.Sprintf(" (%s)", link)
			}
			lines = append(lines, line)
		}
		if len(lines) > 0 {
			sections = append(sections, "Organic Results:\n"+strings.Join(lines, "\n"))
		}
	}

	if len(sections) == 0 {
		return "Không tìm thấy thông tin hữu ích."
	}

	return strings.Join(sections, "\n\n")
}

func ScrapeWeb(targetURL string) string {
	fmt.Printf("📄 [Tool] Đang đọc nội dung web: %s...\n", targetURL)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(targetURL)
	if err != nil {
		return fmt.Sprintf("Lỗi truy cập URL: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("Lỗi: HTTP Status %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("Lỗi đọc nội dung: %v", err)
	}

	// Xử lý thô: Lấy text bên trong các thẻ p, h1, h2
	content := string(body)
	re := regexp.MustCompile(`(?s)<(?:p|h1|h2|h3|li)[^>]*>(.*?)</(?:p|h1|h2|h3|li)>`)
	matches := re.FindAllStringSubmatch(content, -1)

	var textLines []string
	for _, m := range matches {
		cleanText := strings.TrimSpace(stripHTML(m[1]))
		if len(cleanText) > 20 { // Bỏ qua text quá ngắn/rác
			textLines = append(textLines, cleanText)
		}
	}

	finalText := strings.Join(textLines, "\n")
	if len(finalText) > 8000 {
		finalText = finalText[:8000] + "... [Nội dung bị cắt bớt]"
	}
	if finalText == "" {
		return "Không trích xuất được nội dung văn bản hữu ích."
	}
	return finalText
}

func stripHTML(input string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(input, "")
}

func Calculate(expression string) string {
	fmt.Printf("🧮 [Tool] Đang tính toán: %s...\n", expression)
	// Để đơn giản và an toàn trong Go mà không dùng thư viện ngoài phức tạp:
	// Chúng ta sẽ nhờ Gemini tính toán, nhưng giả lập bằng cách trả về thông báo
	// (Hoặc nếu muốn thật, có thể dùng một parser đơn giản, nhưng ở đây ta mô phỏng flow)
	return fmt.Sprintf("Kết quả tính toán cho '%s' (Mô phỏng): Dữ liệu đã được xử lý.", expression)
}

func NeedsRealtimeData(query string) bool {
	normalized := normalizeTextForMatching(query)
	return containsAny(
		normalized,
		"hom nay", "hien tai", "moi nhat",
		"latest", "real-time", "realtime", "giá cổ phiếu", "gia co phieu", "stock price",
		"hom qua", "hom kia", "thu 2", "thu 3", "thu 4", "thu 5", "thu 6", "thu 7", "thu hai", "thu ba", "thu tu", "thu nam", "thu sau", "thu bay",
		"chu nhat", "chủ nhật", "cn",
	)
}

func BuildMarketQueryPlan(query string) MarketQueryPlan {
	normalized := strings.TrimSpace(query)
	if normalized == "" {
		return MarketQueryPlan{}
	}

	now := time.Now()
	tickerHint := ""
	if strings.Contains(strings.ToUpper(normalized), "FPT") {
		tickerHint = " FPT"
	}

	if explicitDate, ok := extractExplicitDate(normalized); ok {
		return MarketQueryPlan{
			SearchQuery: fmt.Sprintf("%s HOSE%s", normalized, tickerHint),
			TimeNote:    fmt.Sprintf("Yêu cầu đã chỉ rõ ngày cụ thể: %s.", explicitDate),
		}
	}

	if resolvedDate, label, ok := resolveTemporalReference(normalized, now); ok {
		dateStr := resolvedDate.Format("02/01/2006")
		return MarketQueryPlan{
			SearchQuery: fmt.Sprintf("%s ngày %s HOSE%s", normalized, dateStr, tickerHint),
			TimeNote:    fmt.Sprintf("Yêu cầu thời gian đã được resolve: %s = %s (%s).", label, utils.TranslateWeekday(resolvedDate.Weekday()), dateStr),
		}
	}

	dateStr := now.Format("02/01/2006")
	return MarketQueryPlan{
		SearchQuery: fmt.Sprintf("%s %s phiên gần nhất HOSE%s", normalized, dateStr, tickerHint),
		TimeNote:    fmt.Sprintf("Yêu cầu dùng dữ liệu mới nhất tính đến thời điểm hiện tại: %s (%s).", utils.TranslateWeekday(now.Weekday()), dateStr),
	}
}

func containsAny(text string, terms ...string) bool {
	for _, term := range terms {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
}

func resolveTemporalReference(query string, now time.Time) (time.Time, string, bool) {
	normalized := normalizeTextForMatching(query)

	switch {
	case containsAny(normalized, "hom nay"):
		return now, "hôm nay", true
	case containsAny(normalized, "hom qua"):
		return now.AddDate(0, 0, -1), "hôm qua", true
	case containsAny(normalized, "hom kia"):
		return now.AddDate(0, 0, -2), "hôm kia", true
	}

	weekdayPatterns := []struct {
		label   string
		matches []string
		weekday time.Weekday
	}{
		{"hôm thứ 2", []string{"hom thu 2", "thu 2", "thu hai"}, time.Monday},
		{"hôm thứ 3", []string{"hom thu 3", "thu 3", "thu ba"}, time.Tuesday},
		{"hôm thứ 4", []string{"hom thu 4", "thu 4", "thu tu"}, time.Wednesday},
		{"hôm thứ 5", []string{"hom thu 5", "thu 5", "thu nam"}, time.Thursday},
		{"hôm thứ 6", []string{"hom thu 6", "thu 6", "thu sau"}, time.Friday},
		{"hôm thứ 7", []string{"hom thu 7", "thu 7", "thu bay"}, time.Saturday},
		{"chủ nhật", []string{"chu nhat", "cn"}, time.Sunday},
	}

	for _, pattern := range weekdayPatterns {
		if containsAny(normalized, pattern.matches...) {
			delta := (int(now.Weekday()) - int(pattern.weekday) + 7) % 7
			resolved := now.AddDate(0, 0, -delta)
			return resolved, pattern.label, true
		}
	}

	return time.Time{}, "", false
}

func extractExplicitDate(query string) (string, bool) {
	re := regexp.MustCompile(`\b\d{1,2}/\d{1,2}/\d{4}\b`)
	match := re.FindString(query)
	if match == "" {
		return "", false
	}
	return match, true
}

func matchesAnyPattern(text string, patterns ...string) bool {
	for _, pattern := range patterns {
		if regexp.MustCompile(pattern).MatchString(text) {
			return true
		}
	}
	return false
}

func normalizeTextForMatching(text string) string {
	replacer := strings.NewReplacer(
		"à", "a", "á", "a", "ạ", "a", "ả", "a", "ã", "a",
		"â", "a", "ầ", "a", "ấ", "a", "ậ", "a", "ẩ", "a", "ẫ", "a",
		"ă", "a", "ằ", "a", "ắ", "a", "ặ", "a", "ẳ", "a", "ẵ", "a",
		"è", "e", "é", "e", "ẹ", "e", "ẻ", "e", "ẽ", "e",
		"ê", "e", "ề", "e", "ế", "e", "ệ", "e", "ể", "e", "ễ", "e",
		"ì", "i", "í", "i", "ị", "i", "ỉ", "i", "ĩ", "i",
		"ò", "o", "ó", "o", "ọ", "o", "ỏ", "o", "õ", "o",
		"ô", "o", "ồ", "o", "ố", "o", "ộ", "o", "ổ", "o", "ỗ", "o",
		"ơ", "o", "ờ", "o", "ớ", "o", "ợ", "o", "ở", "o", "ỡ", "o",
		"ù", "u", "ú", "u", "ụ", "u", "ủ", "u", "ũ", "u",
		"ư", "u", "ừ", "u", "ứ", "u", "ự", "u", "ử", "u", "ữ", "u",
		"ỳ", "y", "ý", "y", "ỵ", "y", "ỷ", "y", "ỹ", "y",
		"đ", "d",
		"À", "a", "Á", "a", "Ạ", "a", "Ả", "a", "Ã", "a",
		"Â", "a", "Ầ", "a", "Ấ", "a", "Ậ", "a", "Ẩ", "a", "Ẫ", "a",
		"Ă", "a", "Ằ", "a", "Ắ", "a", "Ặ", "a", "Ẳ", "a", "Ẵ", "a",
		"È", "e", "É", "e", "Ẹ", "e", "Ẻ", "e", "Ẽ", "e",
		"Ê", "e", "Ề", "e", "Ế", "e", "Ệ", "e", "Ể", "e", "Ễ", "e",
		"Ì", "i", "Í", "i", "Ị", "i", "Ỉ", "i", "Ĩ", "i",
		"Ò", "o", "Ó", "o", "Ọ", "o", "Ỏ", "o", "Õ", "o",
		"Ô", "o", "Ồ", "o", "Ố", "o", "Ộ", "o", "Ổ", "o", "Ỗ", "o",
		"Ơ", "o", "Ờ", "o", "Ớ", "o", "Ợ", "o", "Ở", "o", "Ỡ", "o",
		"Ù", "u", "Ú", "u", "Ụ", "u", "Ủ", "u", "Ũ", "u",
		"Ư", "u", "Ừ", "u", "Ứ", "u", "Ự", "u", "Ử", "u", "Ữ", "u",
		"Ỳ", "y", "Ý", "y", "Ỵ", "y", "Ỷ", "y", "Ỹ", "y",
		"Đ", "d",
	)

	normalized := strings.ToLower(strings.TrimSpace(replacer.Replace(text)))
	return strings.Join(strings.Fields(normalized), " ")
}

func GetCurrentTime() string {
	now := time.Now()
	return fmt.Sprintf("Ngày hiện tại: %s, %s (Múi giờ: Asia/Ho_Chi_Minh)",
		now.Format("02/01/2006 15:04:05"),
		utils.TranslateWeekday(now.Weekday()))
}

func LoadDocument(docType, name string) string {
	return LoadDocumentWithMetadata(docType, name).Content
}

func LoadDocumentWithMetadata(docType, name string) LoadedDocument {
	repoRawRoot := "https://raw.githubusercontent.com/anthropics/financial-services/main/plugins"
	docType = strings.ToLower(strings.TrimSpace(docType))
	name = strings.Trim(strings.TrimSpace(name), "/")

	docURL, err := resolveDocumentURL(repoRawRoot, docType, name)
	if err != nil {
		return LoadedDocument{
			DocType:   docType,
			Name:      name,
			SourceURL: docURL,
			Content:   err.Error(),
		}
	}

	fmt.Printf("📚 [Tool] Đang nạp %s từ GitHub: %s...\n", docType, name)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(docURL)
	if err != nil {
		return LoadedDocument{
			DocType:   docType,
			Name:      name,
			SourceURL: docURL,
			Content:   fmt.Sprintf("Lỗi kết nối khi nạp tài liệu: %v", err),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return LoadedDocument{
			DocType:   docType,
			Name:      name,
			SourceURL: docURL,
			Content:   fmt.Sprintf("Lỗi: Không tìm thấy tài liệu %s mang tên %s trên GitHub (Status: %d).", docType, name, resp.StatusCode),
		}
	}

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return LoadedDocument{
			DocType:   docType,
			Name:      name,
			SourceURL: docURL,
			Content:   fmt.Sprintf("Lỗi khi đọc nội dung tài liệu: %v", err),
		}
	}

	return LoadedDocument{
		DocType:     docType,
		Name:        name,
		SourceURL:   docURL,
		Content:     string(content),
		ContentSize: len(content),
	}
}

func LoadRoutingGuide() string {
	const readmeURL = "https://raw.githubusercontent.com/anthropics/financial-services/main/README.md"

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(readmeURL)
	if err != nil {
		return fmt.Sprintf("Lỗi kết nối khi nạp routing guide: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("Lỗi: Không tải được routing guide từ GitHub (Status: %d).", resp.StatusCode)
	}

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("Lỗi khi đọc routing guide: %v", err)
	}
	return string(content)
}

func resolveDocumentURL(repoRawRoot, docType, name string) (string, error) {
	switch docType {
	case "agent":
		if name == "" {
			return "", fmt.Errorf("Lỗi: Tên agent đang trống.")
		}
		slugParts := strings.Split(name, "/")
		slug := slugParts[len(slugParts)-1]
		return fmt.Sprintf("%s/agent-plugins/%s/agents/%s.md", repoRawRoot, slug, slug), nil
	case "skill", "skills":
		parts := strings.Split(name, "/")
		if len(parts) != 2 {
			return "", fmt.Errorf("Lỗi: Tên skill phải có dạng <agent>/<skill>, ví dụ earnings-reviewer/earnings-analysis.")
		}
		return fmt.Sprintf("%s/agent-plugins/%s/skills/%s/SKILL.md", repoRawRoot, parts[0], parts[1]), nil
	default:
		return "", fmt.Errorf("Lỗi: Loại tài liệu không hợp lệ: %s", docType)
	}
}
