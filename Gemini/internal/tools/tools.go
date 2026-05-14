package tools

import (
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
	return string(body)
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
	repoRoot := `C:\Users\Rabuno\Documents\Repository\financial-services`
	// Import "path/filepath" if needed, but let's use strings for now or add it back
	path := repoRoot + "\\"
	if docType == "agent" {
		path += "agents\\" + name + ".md"
	} else {
		path += "skills\\" + name + "\\SKILL.md"
	}
	fmt.Printf("📚 [Tool] Đang nạp %s: %s...\n", docType, name)
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("Lỗi: Không tìm thấy tài liệu %s mang tên %s.", docType, name)
	}
	return string(content)
}
