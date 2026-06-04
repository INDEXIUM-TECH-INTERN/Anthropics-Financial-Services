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
	
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return fmt.Sprintf("LỖI_TRUY_CẬP: Không thể tạo yêu cầu cho URL: %v", err)
	}
	
	// Thêm User-Agent để tránh bị một số trang chặn
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("LỖI_TRUY_CẬP: Không thể kết nối tới URL: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("LỖI_TRUY_CẬP: HTTP Status %d - Trang web từ chối truy cập hoặc không tồn tại.", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("LỖI_TRUY_CẬP: Không thể đọc nội dung: %v", err)
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
		return "LỖI_TRUY_CẬP: Không trích xuất được nội dung văn bản hữu ích từ trang này."
	}
	return finalText
}

func StripHTML(input string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(input, "")
}
