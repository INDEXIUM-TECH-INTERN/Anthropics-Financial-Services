// Package parser provides pure-Go file parsing for common document formats.
// Replaces the Python markitdown dependency.
//
// Supported formats:
//   - PDF:  text extraction via github.com/ledongthuc/pdf
//   - DOCX: text extraction
//   - XLSX: text extraction via github.com/xuri/excelize/v2
//   - PPTX: text extraction
package parser

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/nguyenthenguyen/docx"
	"github.com/ledongthuc/pdf"
	"github.com/xuri/excelize/v2"
)

// ParseFile đọc file và trả về nội dung text.
// Hỗ trợ: .txt, .md, .json, .csv, .xml, .pdf, .docx, .xlsx, .pptx
func ParseFile(filePath string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".txt", ".md", ".json", ".csv", ".xml", ".html", ".css", ".js", ".ts",
		".go", ".py", ".sql", ".yaml", ".yml", ".log", ".sh", ".ps1":
		return parseTextFile(filePath)

	case ".pdf":
		return parsePDF(filePath)

	case ".docx":
		return parseDOCX(filePath)

	case ".xlsx", ".xls":
		return parseXLSX(filePath)

	case ".pptx":
		return parsePPTX(filePath)

	default:
		// Thử đọc như text trước
		if content, err := parseTextFile(filePath); err == nil && isPrintable(content) {
			return content, nil
		}
		return "", fmt.Errorf("định dạng file không được hỗ trợ: %s", ext)
	}
}

// ParseBytes đọc từ byte slice và trả về nội dung text.
func ParseBytes(data []byte, filename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))

	// Tạo temp file vì một số parser cần đọc từ đường dẫn
	tmpFile, err := os.CreateTemp("", "parse_*"+ext)
	if err != nil {
		return "", fmt.Errorf("lỗi tạo temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("lỗi ghi temp file: %w", err)
	}
	tmpFile.Close()

	return ParseFile(tmpFile.Name())
}

// ─── Internal parsers ───────────────────────────────────────────────

func parseTextFile(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("lỗi đọc file: %w", err)
	}
	return string(data), nil
}

func parsePDF(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("lỗi mở PDF: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("lỗi đọc thông tin file: %w", err)
	}

	pdfReader, err := pdf.NewReader(f, info.Size())
	if err != nil {
		return "", fmt.Errorf("lỗi tạo PDF reader: %w", err)
	}

	var buf strings.Builder
	numPages := pdfReader.NumPage()
	for i := 1; i <= numPages; i++ {
		page := pdfReader.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		buf.WriteString(text)
		buf.WriteString("\n\n")
	}

	result := strings.TrimSpace(buf.String())
	if result == "" {
		return "", fmt.Errorf("PDF không có nội dung text trích xuất được")
	}
	return result, nil
}

func parseDOCX(filePath string) (string, error) {
	doc, err := docx.ReadDocxFile(filePath)
	if err != nil {
		return "", fmt.Errorf("lỗi mở DOCX: %w", err)
	}
	defer doc.Close()

	content := doc.Editable().GetContent()
	text := stripXMLTags(content)
	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("DOCX không có nội dung text")
	}
	return text, nil
}

func parseXLSX(filePath string) (string, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return "", fmt.Errorf("lỗi mở XLSX: %w", err)
	}
	defer f.Close()

	var buf strings.Builder
	sheetList := f.GetSheetList()
	for _, sheetName := range sheetList {
		buf.WriteString(fmt.Sprintf("=== Sheet: %s ===\n", sheetName))
		rows, err := f.GetRows(sheetName)
		if err != nil {
			continue
		}
		for _, row := range rows {
			buf.WriteString(strings.Join(row, "\t"))
			buf.WriteString("\n")
		}
		buf.WriteString("\n")
	}

	result := strings.TrimSpace(buf.String())
	if result == "" {
		return "", fmt.Errorf("XLSX không có nội dung")
	}
	return result, nil
}

func parsePPTX(filePath string) (string, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return "", fmt.Errorf("lỗi mở PPTX: %w", err)
	}
	defer r.Close()

	var allText strings.Builder
	slideNum := 0

	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "ppt/slides/slide") && strings.HasSuffix(f.Name, ".xml") {
			slideNum++
			rc, err := f.Open()
			if err != nil {
				continue
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				continue
			}
			text := extractPPTXText(data)
			if text != "" {
				allText.WriteString(fmt.Sprintf("=== Slide %d ===\n", slideNum))
				allText.WriteString(text)
				allText.WriteString("\n\n")
			}
		}
	}

	result := strings.TrimSpace(allText.String())
	if result == "" {
		return "", fmt.Errorf("PPTX không có nội dung text")
	}
	return result, nil
}

// ─── PPTX XML parsing ──────────────────────────────────────────────

type ptxBody struct {
	XMLName xml.Name  `xml:"http://schemas.openxmlformats.org/presentationml/2006/main txBody"`
	Paras   []ptxPara `xml:"http://schemas.openxmlformats.org/drawingml/2006/main p"`
}

type ptxPara struct {
	XMLName xml.Name    `xml:"http://schemas.openxmlformats.org/drawingml/2006/main p"`
	Runs    []ptxText   `xml:"http://schemas.openxmlformats.org/drawingml/2006/main t"`
}

type ptxText struct {
	XMLName xml.Name `xml:"http://schemas.openxmlformats.org/drawingml/2006/main t"`
	Text    string   `xml:",chardata"`
}

func extractPPTXText(data []byte) string {
	var body ptxBody
	if err := xml.Unmarshal(data, &body); err != nil {
		return stripXMLTags(string(data))
	}

	var texts []string
	for _, para := range body.Paras {
		var line strings.Builder
		for _, t := range para.Runs {
			line.WriteString(t.Text)
		}
		if s := strings.TrimSpace(line.String()); s != "" {
			texts = append(texts, s)
		}
	}
	return strings.Join(texts, "\n")
}

// ─── Helpers ────────────────────────────────────────────────────────

func stripXMLTags(s string) string {
	var buf strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
			buf.WriteRune(' ')
		case !inTag:
			buf.WriteRune(r)
		}
	}
	result := strings.Join(strings.Fields(buf.String()), " ")
	return result
}

func isPrintable(s string) bool {
	if len(s) == 0 {
		return false
	}
	printable := 0
	for _, r := range s {
		if r >= 32 && r < 127 || r == '\n' || r == '\r' || r == '\t' {
			printable++
		}
	}
	return float64(printable)/float64(len(s)) > 0.85
}

// Truncate cắt text nếu quá dài
func Truncate(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	half := maxLen / 2
	return text[:half] + "\n\n... [Nội dung bị cắt bớt — quá dài để hiển thị đầy đủ] ...\n\n" + text[len(text)-half:]
}
