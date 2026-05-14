package tools

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

// --- Các hàm Tool (MCP-style) ---

func SearchGoogle(query string) string {
	apiKey := os.Getenv("SERPAPI_KEY")
	if apiKey == "" {
		return "Lỗi: Chưa cấu hình SERPAPI_KEY."
	}
	fmt.Printf("🌐 [Tool] Đang tìm kiếm: %s...\n", query)
	url := fmt.Sprintf("https://serpapi.com/search?engine=google&q=%s&api_key=%s", strings.ReplaceAll(query, " ", "+"), apiKey)
	resp, err := http.Get(url)
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

	// Lấy Answer Box (Nếu có - thường chứa giá cổ phiếu trực tiếp)
	summary := ""
	if ab, ok := data["answer_box"].(map[string]interface{}); ok {
		summary += fmt.Sprintf("Answer Box: %v\n", ab)
	}

	// Lấy các kết quả hữu cơ đầu tiên
	if results, ok := data["organic_results"].([]interface{}); ok {
		for i, res := range results {
			if i >= 3 { // Chỉ lấy 3 kết quả đầu
				break
			}
			r := res.(map[string]interface{})
			summary += fmt.Sprintf("- %s: %s\n", r["title"], r["snippet"])
		}
	}

	if summary == "" {
		return "Không tìm thấy thông tin hữu ích."
	}

	return summary
}

func GetCurrentTime() string {
	now := time.Now()
	return fmt.Sprintf("Ngày hiện tại: %s, %s (Múi giờ: Asia/Ho_Chi_Minh)", 
		now.Format("02/01/2006 15:04:05"), 
		translateWeekday(now.Weekday()))
}

func translateWeekday(wd time.Weekday) string {
	days := map[time.Weekday]string{
		time.Sunday:    "Chủ Nhật",
		time.Monday:    "Thứ Hai",
		time.Tuesday:   "Thứ Ba",
		time.Wednesday: "Thứ Tư",
		time.Thursday:  "Thứ Năm",
		time.Friday:    "Thứ Sáu",
		time.Saturday:  "Thứ Bảy",
	}
	return days[wd]
}

func LoadDocument(docType, name string) string {
	repoRawRoot := "https://raw.githubusercontent.com/anthropics/financial-services/main"
	var url string
	if docType == "agent" {
		url = fmt.Sprintf("%s/agents/%s.md", repoRawRoot, name)
	} else {
		url = fmt.Sprintf("%s/skills/%s/SKILL.md", repoRawRoot, name)
	}

	fmt.Printf("📚 [Tool] Đang nạp %s từ GitHub: %s...\n", docType, name)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Sprintf("Lỗi kết nối khi nạp tài liệu: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("Lỗi: Không tìm thấy tài liệu %s mang tên %s trên GitHub (Status: %d).", docType, name, resp.StatusCode)
	}

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("Lỗi khi đọc nội dung tài liệu: %v", err)
	}
	return string(content)
}
