package utils

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	scripts_parser "gemini-cli/internal/scripts/parser"
)

// ParseAttachment cố gắng chuyển đổi file đính kèm thành văn bản để LLM có thể đọc trực tiếp.
// Sử dụng Microsoft markitdown qua python script.
func ParseAttachment(name, mimeType, dataBase64 string) (string, bool) {
	data, err := base64.StdEncoding.DecodeString(dataBase64)
	if err != nil {
		return fmt.Sprintf("[Lỗi giải mã Base64: %v]", err), false
	}

	ext := strings.ToLower(filepath.Ext(name))

	// 1. Nếu là ảnh, video, âm thanh -> Bỏ qua, để Gemini tự xử lý qua InlineData nếu hỗ trợ
	if strings.HasPrefix(mimeType, "image/") || strings.HasPrefix(mimeType, "video/") || strings.HasPrefix(mimeType, "audio/") {
		return "", false
	}

	// 2. Xử lý các loại file text phổ biến (đọc trực tiếp cho nhanh)
	textExts := []string{".txt", ".md", ".json", ".xml", ".go", ".py", ".js", ".ts", ".html", ".css", ".sql", ".yaml", ".yml"}
	for _, e := range textExts {
		if ext == e {
			return string(data), true
		}
	}
	if ext == ".csv" || mimeType == "text/csv" || strings.HasPrefix(mimeType, "text/") {
		return string(data), true
	}

	// 3. Sử dụng markitdown cho các file phức tạp (Excel, Word, PDF, PPTX...)
	return ParseWithMarkItDown(name, data)
}

// ParseWithMarkItDown ghi dữ liệu ra file tạm và parse bằng Go parser.
// Thay thế cho Python markitdown script — không cần Python runtime.
func ParseWithMarkItDown(filename string, data []byte) (string, bool) {
	// Tạo file tạm
	tmpFile, err := os.CreateTemp("", "parser_*"+filepath.Ext(filename))
	if err != nil {
		return fmt.Sprintf("[Lỗi tạo file tạm: %v]", err), false
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Sprintf("[Lỗi ghi file tạm: %v]", err), false
	}
	tmpFile.Close()

	// Dùng Go parser thay vì Python script
	content, err := scripts_parser.ParseFile(tmpFile.Name())
	if err != nil {
		return fmt.Sprintf("[Lỗi phân tích file: %v]", err), false
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return "[File không có nội dung văn bản nào được trích xuất]", true
	}

	return content, true
}

// GetFileContentWrapper bọc nội dung file vào một block Markdown rõ ràng cho LLM
func GetFileContentWrapper(name, content string) string {
	return fmt.Sprintf("\n--- BẮT ĐẦU NỘI DUNG FILE: %s ---\n%s\n--- KẾT THÚC NỘI DUNG FILE: %s ---\n", name, content, name)
}
