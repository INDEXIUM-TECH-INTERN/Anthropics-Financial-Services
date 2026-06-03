package scraper

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"
)

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
		cleanText := strings.TrimSpace(StripHTML(m[1]))
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

func StripHTML(input string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(input, "")
}
