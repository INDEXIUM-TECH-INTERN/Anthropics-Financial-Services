package market

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"gemini-cli/internal/utils"
)

var yahooBaseURL = "https://query1.finance.yahoo.com/v8/finance/chart"
var ddgBaseURL = "https://html.duckduckgo.com/html"

type MarketQueryPlan struct {
	SearchQuery string
	TimeNote    string
}

func SearchTavily(query string) string {
	apiKey := os.Getenv("TAVILY_API_KEY")
	if apiKey == "" {
		return ExecuteFreeSearchFallback(query)
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
		return ExecuteFreeSearchFallback(query)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return ExecuteFreeSearchFallback(query)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ExecuteFreeSearchFallback(query)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return ExecuteFreeSearchFallback(query)
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
		return ExecuteFreeSearchFallback(query)
	}

	return strings.Join(sections, "\n\n")
}

func SearchGoogle(query string) string {
	apiKey := os.Getenv("SERPAPI_KEY")
	if apiKey == "" {
		return ExecuteFreeSearchFallback(query)
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
		return ExecuteFreeSearchFallback(query)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ExecuteFreeSearchFallback(query)
	}

	// Giải mã JSON để lọc thông tin quan trọng
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return ExecuteFreeSearchFallback(query)
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
		return ExecuteFreeSearchFallback(query)
	}

	return strings.Join(sections, "\n\n")
}

func NeedsRealtimeData(query string) bool {
	normalized := normalizeTextForMatching(query)
	return utils.ContainsAny(
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

func resolveTemporalReference(query string, now time.Time) (time.Time, string, bool) {
	normalized := normalizeTextForMatching(query)

	switch {
	case utils.ContainsAny(normalized, "hom nay"):
		return now, "hôm nay", true
	case utils.ContainsAny(normalized, "hom qua"):
		return now.AddDate(0, 0, -1), "hôm qua", true
	case utils.ContainsAny(normalized, "hom kia"):
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
		if utils.ContainsAny(normalized, pattern.matches...) {
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

type YahooFinanceResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Symbol             string  `json:"symbol"`
				ExchangeName       string  `json:"exchangeName"`
				RegularMarketPrice float64 `json:"regularMarketPrice"`
				ChartPreviousClose float64 `json:"chartPreviousClose"`
				PreviousClose      float64 `json:"previousClose"`
				Currency           string  `json:"currency"`
			} `json:"meta"`
		} `json:"result"`
		Error interface{} `json:"error"`
	} `json:"chart"`
}

func extractTicker(query string) (string, bool) {
	re := regexp.MustCompile(`(?i)\b(FPT|VNM|HPG|HDB|ACB|VIC|VHM|VRE|MSN|MWG|TCB|VPB|MBB|STB|CTG|BID|VCB|GAS|PLX|POW|SAB|EIB|LPB|VIB|SHB|HAG|DXG|DIG|NLG|KDH|PVD|PVS|SSI|VND)(?:\.VN)?\b`)
	match := re.FindStringSubmatch(query)
	if len(match) > 1 {
		return strings.ToUpper(match[1]), true
	}
	return "", false
}

func FetchYahooFinanceDataFallback(ticker string) string {
	ticker = strings.ToUpper(ticker)
	urlVN := fmt.Sprintf("%s/%s.VN", yahooBaseURL, ticker)
	
	result, err := queryYahooAPI(urlVN, ticker)
	if err == nil && result != "" {
		return result
	}
	
	urlPlain := fmt.Sprintf("%s/%s", yahooBaseURL, ticker)
	result, err = queryYahooAPI(urlPlain, ticker)
	if err == nil && result != "" {
		return result
	}
	
	return "Lỗi: Không thể lấy dữ liệu từ Yahoo Finance cho mã " + ticker + "."
}

func queryYahooAPI(apiURL, ticker string) (string, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36")
	
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP status %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	
	var data YahooFinanceResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}
	
	if len(data.Chart.Result) == 0 {
		return "", fmt.Errorf("empty chart result")
	}
	
	meta := data.Chart.Result[0].Meta
	
	previousClose := meta.ChartPreviousClose
	if previousClose == 0 {
		previousClose = meta.PreviousClose
	}
	
	priceDiff := meta.RegularMarketPrice - previousClose
	var priceMovement string
	if previousClose > 0 {
		percent := (priceDiff / previousClose) * 100
		if priceDiff > 0 {
			priceMovement = fmt.Sprintf("+%.2f (+%.2f%%)", priceDiff, percent)
		} else if priceDiff < 0 {
			priceMovement = fmt.Sprintf("%.2f (%.2f%%)", priceDiff, percent)
		} else {
			priceMovement = "0.00 (0.00%)"
		}
	} else {
		priceMovement = "0.00 (0.00%)"
	}
	
	var lines []string
	symbol := meta.Symbol
	if symbol == "" {
		symbol = ticker
	}
	lines = append(lines, fmt.Sprintf("stock: %s", symbol))
	
	exchange := meta.ExchangeName
	if exchange == "" {
		exchange = "HOSE"
	}
	lines = append(lines, fmt.Sprintf("exchange: %s", exchange))
	lines = append(lines, fmt.Sprintf("price: %.2f", meta.RegularMarketPrice))
	
	currency := meta.Currency
	if currency == "" {
		currency = "VND"
	}
	lines = append(lines, fmt.Sprintf("currency: %s", currency))
	lines = append(lines, fmt.Sprintf("price_movement: %s", priceMovement))
	
	return "Stock Information:\n" + strings.Join(lines, "\n"), nil
}

func stripHTML(input string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	result := re.ReplaceAllString(input, "")
	result = strings.ReplaceAll(result, "&amp;", "&")
	result = strings.ReplaceAll(result, "&lt;", "<")
	result = strings.ReplaceAll(result, "&gt;", ">")
	result = strings.ReplaceAll(result, "&quot;", "\"")
	result = strings.ReplaceAll(result, "&#x27;", "'")
	result = strings.ReplaceAll(result, "&#39;", "'")
	result = strings.ReplaceAll(result, "&nbsp;", " ")
	return strings.TrimSpace(result)
}

func cleanDDGURL(rawURL string) string {
	if strings.Contains(rawURL, "uddg=") {
		u, err := url.Parse(rawURL)
		if err == nil {
			realURL := u.Query().Get("uddg")
			if realURL != "" {
				return realURL
			}
		}
	}
	if strings.HasPrefix(rawURL, "//") {
		return "https:" + rawURL
	}
	if strings.HasPrefix(rawURL, "/") {
		return "https://duckduckgo.com" + rawURL
	}
	return rawURL
}

func SearchDuckDuckGoFree(query string) string {
	fmt.Printf("🌐 [Tool] Đang tìm kiếm (DuckDuckGo Free): %s...\n", query)
	ddgURL := fmt.Sprintf("%s/?q=%s", ddgBaseURL, url.QueryEscape(query))
	
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", ddgURL, nil)
	if err != nil {
		return "Lỗi khởi tạo request tìm kiếm."
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36")
	
	resp, err := client.Do(req)
	if err != nil {
		return "Lỗi kết nối tìm kiếm."
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("Lỗi kết nối tìm kiếm (HTTP %d).", resp.StatusCode)
	}
	
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "Lỗi đọc kết quả tìm kiếm."
	}
	
	htmlContent := string(bodyBytes)
	
	blocks := strings.Split(htmlContent, `<div class="result`)
	if len(blocks) <= 1 {
		blocks = strings.Split(htmlContent, `class="result`)
	}
	
	var lines []string
	count := 0
	
	reTitle := regexp.MustCompile(`<a\s+class="result__a"\s+href="([^"]+)"[^>]*>([\s\S]*?)</a>`)
	reSnippet := regexp.MustCompile(`class="[^"]*result__snippet[^"]*"[^>]*>([\s\S]*?)</`)
	
	for i := 1; i < len(blocks); i++ {
		block := blocks[i]
		
		titleMatches := reTitle.FindStringSubmatch(block)
		if len(titleMatches) < 3 {
			continue
		}
		rawURL := titleMatches[1]
		title := stripHTML(titleMatches[2])
		if title == "" {
			continue
		}
		
		snippet := ""
		snippetMatches := reSnippet.FindStringSubmatch(block)
		if len(snippetMatches) >= 2 {
			snippet = stripHTML(snippetMatches[1])
		}
		
		link := cleanDDGURL(rawURL)
		
		if len(snippet) > 300 {
			snippet = snippet[:300] + "..."
		}
		
		line := fmt.Sprintf("- %s [URL: %s]", title, link)
		if snippet != "" {
			line += fmt.Sprintf(": %s", snippet)
		}
		lines = append(lines, line)
		
		count++
		if count >= 5 {
			break
		}
	}
	
	if len(lines) == 0 {
		allTitles := reTitle.FindAllStringSubmatch(htmlContent, -1)
		allSnippets := reSnippet.FindAllStringSubmatch(htmlContent, -1)
		for idx, tMatch := range allTitles {
			if idx >= 5 {
				break
			}
			rawURL := tMatch[1]
			title := stripHTML(tMatch[2])
			if title == "" {
				continue
			}
			snippet := ""
			if idx < len(allSnippets) {
				snippet = stripHTML(allSnippets[idx][1])
			}
			link := cleanDDGURL(rawURL)
			if len(snippet) > 300 {
				snippet = snippet[:300] + "..."
			}
			line := fmt.Sprintf("- %s [URL: %s]", title, link)
			if snippet != "" {
				line += fmt.Sprintf(": %s", snippet)
			}
			lines = append(lines, line)
		}
	}
	
	if len(lines) == 0 {
		return "Không tìm thấy kết quả tìm kiếm nào từ DuckDuckGo."
	}
	
	return "Search Results:\n" + strings.Join(lines, "\n")
}

func ExecuteFreeSearchFallback(query string) string {
	var results []string
	if ticker, ok := extractTicker(query); ok {
		yahooData := FetchYahooFinanceDataFallback(ticker)
		if yahooData != "" && !strings.HasPrefix(yahooData, "Lỗi:") {
			results = append(results, yahooData)
		}
	}
	ddgData := SearchDuckDuckGoFree(query)
	if ddgData != "" && !strings.HasPrefix(ddgData, "Không tìm thấy") && !strings.HasPrefix(ddgData, "Lỗi") {
		results = append(results, ddgData)
	}
	
	if len(results) == 0 {
		return "Không tìm thấy kết quả tìm kiếm hoặc dữ liệu tài chính nào."
	}
	return strings.Join(results, "\n\n")
}
