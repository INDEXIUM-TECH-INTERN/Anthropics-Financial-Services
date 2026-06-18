// Package report_gen generates Excel (.xlsx) reports using pure Go.
// For PowerPoint, use the Python script in scripts/report_generator.py.
package report_gen

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// ReportFormat loại báo cáo hỗ trợ
type ReportFormat string

const (
	FormatXLSX ReportFormat = "xlsx"
	FormatPPTX ReportFormat = "pptx"
)

// Generate tạo báo cáo theo format và trả về đường dẫn file output.
// Với xlsx: dataJSON có thể là array (single sheet) hoặc dict (multi-sheet).
// Với pptx: trả về lỗi — dùng Python script report_generator.py cho PPTX.
func Generate(format ReportFormat, title string, dataJSON string, outputPath string) (string, error) {
	var data interface{}
	if dataJSON != "" {
		if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
			return "", fmt.Errorf("lỗi parse JSON data: %w", err)
		}
	}

	outDir := filepath.Dir(outputPath)
	if outDir != "" && outDir != "." {
		if err := os.MkdirAll(outDir, 0755); err != nil {
			return "", fmt.Errorf("lỗi tạo thư mục output: %w", err)
		}
	}

	switch format {
	case FormatXLSX:
		if err := generateXLSX(outputPath, title, data); err != nil {
			return "", err
		}
	case FormatPPTX:
		return "", fmt.Errorf("PPTX generation cần Python script — dùng report_generator.py")
	default:
		return "", fmt.Errorf("format không hợp lệ: %s", format)
	}

	return outputPath, nil
}

// GenerateXLSX là public helper để tạo Excel trực tiếp
func GenerateXLSX(outputPath string, title string, data interface{}) error {
	return generateXLSX(outputPath, title, data)
}

// ─── Excel Generation ───────────────────────────────────────────────

func generateXLSX(outputPath string, title string, data interface{}) error {
	f := excelize.NewFile()
	defer f.Close()

	if data == nil {
		createFallbackSheet(f, title)
		return f.SaveAs(outputPath)
	}

	switch d := data.(type) {
	case map[string]interface{}:
		first := true
		for sheetName, rows := range d {
			sheetName = sanitizeSheetName(sheetName)
			if first {
				f.SetSheetName("Sheet1", sheetName)
				writeSheet(f, sheetName, title, rows)
				first = false
			} else {
				idx, _ := f.NewSheet(sheetName)
				f.SetActiveSheet(idx)
				writeSheet(f, sheetName, title, rows)
			}
		}
	case []interface{}:
		sheetName := sanitizeSheetName(title)
		f.SetSheetName("Sheet1", sheetName)
		writeSheet(f, sheetName, title, d)
	default:
		createFallbackSheet(f, title)
	}

	return f.SaveAs(outputPath)
}

func writeSheet(f *excelize.File, sheetName, sheetTitle string, rows interface{}) {
	// Title
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 16, Color: "1F4E79"},
	})
	f.SetCellValue(sheetName, "A1", sheetTitle)
	f.SetCellStyle(sheetName, "A1", "A1", titleStyle)

	// Header style
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"1F4E79"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "D3D3D3", Style: 1},
			{Type: "right", Color: "D3D3D3", Style: 1},
			{Type: "top", Color: "D3D3D3", Style: 1},
			{Type: "bottom", Color: "D3D3D3", Style: 1},
		},
	})

	dataStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Size: 11},
		Border: []excelize.Border{
			{Type: "left", Color: "D3D3D3", Style: 1},
			{Type: "right", Color: "D3D3D3", Style: 1},
			{Type: "top", Color: "D3D3D3", Style: 1},
			{Type: "bottom", Color: "D3D3D3", Style: 1},
		},
	})

	rowIdx := 3 // Bắt đầu sau title + empty row

	switch r := rows.(type) {
	case []interface{}:
		for _, row := range r {
			switch v := row.(type) {
			case []interface{}:
				for i, col := range v {
					cell, _ := excelize.CoordinatesToCellName(i+1, rowIdx)
					f.SetCellValue(sheetName, cell, col)
					if rowIdx == 3 {
						f.SetCellStyle(sheetName, cell, cell, headerStyle)
					} else {
						f.SetCellStyle(sheetName, cell, cell, dataStyle)
					}
				}
				rowIdx++
			case map[string]interface{}:
				colIdx := 1
				for _, val := range v {
					cell, _ := excelize.CoordinatesToCellName(colIdx, rowIdx)
					f.SetCellValue(sheetName, cell, val)
					f.SetCellStyle(sheetName, cell, cell, dataStyle)
					colIdx++
				}
				rowIdx++
			default:
				cell, _ := excelize.CoordinatesToCellName(1, rowIdx)
				f.SetCellValue(sheetName, cell, v)
				f.SetCellStyle(sheetName, cell, cell, dataStyle)
				rowIdx++
			}
		}
	}

	// Auto-width (best-effort)
	for col := 1; col <= 20; col++ {
		colName, _ := excelize.ColumnNumberToName(col)
		f.SetColWidth(sheetName, colName, colName, 18)
	}
}

func createFallbackSheet(f *excelize.File, title string) {
	sheetName := "Overview"
	f.SetSheetName("Sheet1", sheetName)

	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 18, Color: "1F4E79"},
	})
	labelStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 11},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"F2F2F2"}, Pattern: 1},
	})
	valStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Size: 11},
	})

	f.SetCellValue(sheetName, "A1", title)
	f.SetCellStyle(sheetName, "A1", "A1", titleStyle)

	metadata := [][]interface{}{
		{"Loại báo cáo", "Báo cáo Tài chính Excel"},
		{"Ngày tạo", time.Now().Format("2006-01-02")},
		{"Người tạo", "Indexium Financial AI Agent"},
		{"Mô tả", "Báo cáo tài chính chi tiết được tạo tự động bởi AI."},
	}
	for i, row := range metadata {
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", i+3), row[0])
		f.SetCellStyle(sheetName, fmt.Sprintf("A%d", i+3), fmt.Sprintf("A%d", i+3), labelStyle)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", i+3), row[1])
		f.SetCellStyle(sheetName, fmt.Sprintf("B%d", i+3), fmt.Sprintf("B%d", i+3), valStyle)
	}
	f.SetColWidth(sheetName, "A", "A", 20)
	f.SetColWidth(sheetName, "B", "B", 50)
}

// ─── Helpers ────────────────────────────────────────────────────────

func sanitizeSheetName(name string) string {
	name = strings.TrimSpace(name)
	if len(name) > 31 {
		name = name[:31]
	}
	invalid := []string{":", "\\", "/", "?", "*", "[", "]"}
	for _, ch := range invalid {
		name = strings.ReplaceAll(name, ch, "_")
	}
	if name == "" {
		name = "Sheet"
	}
	return name
}

// IsNumeric kiểm tra chuỗi có phải số không
func IsNumeric(val string) bool {
	if val == "" {
		return false
	}
	if strings.HasPrefix(val, "=") {
		return true
	}
	cleaned := strings.ReplaceAll(strings.ReplaceAll(val, ",", ""), "%", "")
	_, err := strconv.ParseFloat(cleaned, 64)
	return err == nil
}
