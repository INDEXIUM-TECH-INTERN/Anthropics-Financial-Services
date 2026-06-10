package scraper

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// isBlockedURL checks if the URL points to internal/private network addresses.
// This prevents SSRF (Server-Side Request Forgery) attacks.
func isBlockedURL(targetURL string) error {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %v", err)
	}

	// Only allow http and https schemes
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme: %s (only http/https allowed)", parsed.Scheme)
	}

	// Resolve the hostname and check for private/internal IPs
	ips, err := net.LookupIP(parsed.Hostname())
	if err != nil {
		return fmt.Errorf("cannot resolve host: %v", err)
	}

	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return fmt.Errorf("access to internal addresses is blocked")
		}
		// Check private ranges: 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16
		if ip.To4() != nil {
			if ip[0] == 10 || (ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31) || (ip[0] == 192 && ip[1] == 168) {
				return fmt.Errorf("access to private network addresses is blocked")
			}
		}
		// Block IPv6 loopback and link-local
		if ip.To16() != nil && ip.To4() == nil {
			if ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				return fmt.Errorf("access to internal IPv6 addresses is blocked")
			}
		}
	}

	return nil
}

func ScrapeWeb(targetURL string) string {
	fmt.Printf("📄 [Tool] Đang đọc nội dung web: %s...\n", targetURL)

	// SSRF protection: validate URL before making the request
	if err := isBlockedURL(targetURL); err != nil {
		return fmt.Sprintf("LỖI_BẢO_MẬT: %v", err)
	}

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

	body, err := io.ReadAll(resp.Body)
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
