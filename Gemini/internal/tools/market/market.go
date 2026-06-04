package market

import (
	"bytes"
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

func SearchTavily(query string) string {
	apiKey := os.Getenv("TAVILY_API_KEY")
	if apiKey == "" {
		return "Lỗi: Chưa cấu hình TAVILY_API_KEY."
	}
	fmt.Printf("🌐 [Tool] Đang tìm kiếm (Tavily): %s...\n", query)
	
	reqBody, _ := json.Marshal(map[string]interface{}{
		"api_key": apiKey,
		"query": query,
		"search_depth": "advanced",
		"include_answer": true,
		"include_images": false,
		"include_raw_content": false,
		"max_results": 5,
	})

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("POST", "https://api.tavily.com/search", bytes.NewBuffer(reqBody))
	if err != nil {
		return "Lỗi khởi tạo request tìm kiếm."
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "Lỗi kết nối tìm kiếm."
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return "Lỗi giải mã kết quả tìm kiếm."
	}

	var sections []string

	if answer, ok := data["answer"].(string); ok && answer != "" {
		sections = append(sections, "Tavily Answer:\n"+answer)
	}

	if results, ok := data["results"].([]interface{}); ok {
		var lines []string
		for _, res := range results {
			r, ok := res.(map[string]interface{})
			if !ok {
				continue
			}
			title := strings.TrimSpace(fmt.Sprintf("%v", r["title"]))
			if title == "" || title == "<nil>" {
				continue
			}
			content := strings.TrimSpace(fmt.Sprintf("%v", r["content"]))
			if len(content) > 300 {
				content = content[:300] + "..."
			}
			link := strings.TrimSpace(fmt.Sprintf("%v", r["url"]))

			line := fmt.Sprintf("- %s [URL: %s]", title, link)
			if content != "" && content != "<nil>" {
				line += fmt.Sprintf(": %s", content)
			}
			lines = append(lines, line)
		}
		if len(lines) > 0 {
			sections = append(sections, "Search Results:\n"+strings.Join(lines, "\n"))
		}
	}

	if len(sections) == 0 {
		return "Không tìm thấy thông tin hữu ích từ Tavily."
	}

	return strings.Join(sections, "\n\n")
}

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

			line := fmt.Sprintf("- %s [URL: %s] [DATE: %s]", title, link, date)
			if snippet != "" && snippet != "<nil>" {
				line += fmt.Sprintf(": %s", snippet)
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
			SearchQuery: fmt.Sprintf("%s báo cáo tài chính chi tiết ngày %s HOSE%s", normalized, explicitDate, tickerHint),
			TimeNote:    fmt.Sprintf("Yêu cầu đã chỉ rõ ngày cụ thể: %s.", explicitDate),
		}
	}

	if resolvedDate, label, ok := resolveTemporalReference(normalized, now); ok {
		dateStr := resolvedDate.Format("02/01/2006")
		return MarketQueryPlan{
			SearchQuery: fmt.Sprintf("%s tin tức báo cáo chi tiết ngày %s HOSE%s", normalized, dateStr, tickerHint),
			TimeNote:    fmt.Sprintf("Yêu cầu thời gian đã được resolve: %s = %s (%s).", label, utils.TranslateWeekday(resolvedDate.Weekday()), dateStr),
		}
	}

	dateStr := now.Format("02/01/2006")
	return MarketQueryPlan{
		SearchQuery: fmt.Sprintf("%s tin tức báo cáo chi tiết mới nhất %s HOSE%s", normalized, dateStr, tickerHint),
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
