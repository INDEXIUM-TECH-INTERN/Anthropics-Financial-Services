package tools

import (
	"fmt"
	"strings"
	"time"

	"gemini-cli/internal/utils"
)

// ParseTimeExpression phân tích biểu thức thời gian cơ bản
func ParseTimeExpression(expr string) string {
	expr = strings.TrimSpace(expr)
	now := time.Now()
	currentYear := now.Year()

	// Xử lý các pattern cơ bản
	switch {
	case expr == "giờ giao dịch":
		return parseBusinessTime(expr)
	case strings.Contains(expr, "năm"):
		return parseYearExpression(expr, currentYear)
	case strings.Contains(expr, "tháng"):
		return parseMonthExpression(expr, now)
	case strings.Contains(expr, "quý"):
		return parseQuarterExpression(expr, now)
	case strings.Contains(expr, "ngày") || expr == "hôm nay" || expr == "hôm qua" || expr == "ngày mai":
		return parseDayExpression(expr, now)
	default:
		return fmt.Sprintf("⚠️ Không hiểu biểu thức thời gian: %s", expr)
	}
}

// parseYearExpression xử lý các biểu thức liên quan đến năm
func parseYearExpression(expr string, currentYear int) string {
	// Pattern: "3 năm gần nhất", "năm ngoái", "năm tới"
	if strings.Contains(expr, "gần nhất") {
		parts := strings.Split(expr, " ")
		if len(parts) >= 3 && strings.HasPrefix(parts[0], "3") {
			return "📅 3 năm gần nhất: 2024, 2025, 2026"
		}
		if len(parts) >= 3 && strings.HasPrefix(parts[0], "5") {
			return "📅 5 năm gần nhất: 2022, 2023, 2024, 2025, 2026"
		}
	}

	if strings.Contains(expr, "năm ngoái") || strings.Contains(expr, "năm trước") {
		return fmt.Sprintf("📅 Năm ngoái: %d", currentYear-1)
	}

	if strings.Contains(expr, "năm tới") {
		return fmt.Sprintf("📅 Năm tới: %d", currentYear+1)
	}

	return fmt.Sprintf("📅 %s (%d)", expr, currentYear)
}

// parseBusinessTime xử lý các biểu thức kinh doanh
func parseBusinessTime(expr string) string {
	now := time.Now()
	weekday := now.Weekday()
	hour := now.Hour()

	if expr == "giờ giao dịch" {
		if weekday >= time.Monday && weekday <= time.Friday && hour >= 9 && hour < 15 {
			return fmt.Sprintf("🕐 Đang trong giờ giao dịch: %s", now.Format("15:04"))
		}
		return fmt.Sprintf("🕐 Ngoài giờ giao dịch. Giờ tiếp theo: %s",
			time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, time.Local).AddDate(0, 0, int(time.Saturday-weekday)+1).Format("15:04 02/01/2006"))
	}
	return fmt.Sprintf("🕐 %s (%s)", expr, now.Format("15:04"))
}

// parseMonthExpression xử lý các biểu thức liên quan đến tháng
func parseMonthExpression(expr string, now time.Time) string {
	currentYear := now.Year()
	currentMonth := now.Month()

	if strings.Contains(expr, "gần nhất") {
		parts := strings.Split(expr, " ")
		if len(parts) >= 3 && strings.HasPrefix(parts[0], "6") {
			prevMonth := currentMonth - 5
			if prevMonth < 1 {
				prevMonth += 12
				currentYear--
			}
			return fmt.Sprintf("📅 6 tháng gần nhất: %d/%d đến %d/%d",
				prevMonth, currentYear, currentMonth, now.Year())
		}
	}

	if strings.Contains(expr, "tháng trước") {
		prevMonth := currentMonth - 1
		if prevMonth < 1 {
			prevMonth = 12
			currentYear--
		}
		return fmt.Sprintf("📅 Tháng trước: %d/%d", prevMonth, currentYear)
	}

	if strings.Contains(expr, "tháng tới") {
		nextMonth := currentMonth + 1
		if nextMonth > 12 {
			nextMonth = 1
			currentYear++
		}
		return fmt.Sprintf("📅 Tháng tới: %d/%d", nextMonth, currentYear)
	}

	return fmt.Sprintf("📅 %s (%d/%d)", expr, currentMonth, currentYear)
}

// parseQuarterExpression xử lý các biểu thức liên quan đến quý
func parseQuarterExpression(expr string, now time.Time) string {
	currentYear := now.Year()
	quarter := (now.Month()-1)/4 + 1

	if strings.Contains(expr, "gần nhất") {
		return fmt.Sprintf("📅 Quý gần nhất: Q%d/%d và Q%d/%d",
			quarter-1, currentYear, quarter, currentYear)
	}

	if strings.Contains(expr, "quý trước") {
		prevQuarter := quarter - 1
		if prevQuarter < 1 {
			prevQuarter = 4
			currentYear--
		}
		return fmt.Sprintf("📅 Quý trước: Q%d/%d", prevQuarter, currentYear)
	}

	if strings.Contains(expr, "quý tới") {
		nextQuarter := quarter + 1
		if nextQuarter > 4 {
			nextQuarter = 1
			currentYear++
		}
		return fmt.Sprintf("📅 Quý tới: Q%d/%d", nextQuarter, currentYear)
	}

	return fmt.Sprintf("📅 %s (Q%d/%d)", expr, quarter, currentYear)
}

// parseDayExpression xử lý các biểu thức liên quan đến ngày
func parseDayExpression(expr string, now time.Time) string {
	switch expr {
	case "hôm nay":
		return fmt.Sprintf("📅 Hôm nay: %s", now.Format("02/01/2006"))
	case "hôm qua":
		yesterday := now.AddDate(0, 0, -1)
		return fmt.Sprintf("📅 Hôm qua: %s", yesterday.Format("02/01/2006"))
	case "ngày mai":
		tomorrow := now.AddDate(0, 0, 1)
		return fmt.Sprintf("📅 Ngày mai: %s", tomorrow.Format("02/01/2006"))
	}

	// Pattern cho ngày cụ thể: "15/06/2026"
	if len(expr) == 10 && expr[2] == '/' && expr[5] == '/' {
		return fmt.Sprintf("📅 Ngày cụ thể: %s", expr)
	}

	return fmt.Sprintf("📅 %s (%s)", expr, now.Format("02/01/2006"))
}

// NormalizeTimeExpression chuẩn hóa và giải thích biểu thức thời gian
func NormalizeTimeExpression(expr string) string {
	return ParseTimeExpression(expr)
}

// GetCurrentTimeInfoEx trả về thông tin thời gian hiện tại mở rộng
func GetCurrentTimeInfoEx() string {
	now := time.Now()
	currentYear := now.Year()
	currentMonth := now.Month()
	quarter := (now.Month()-1)/4 + 1

	return fmt.Sprintf(`🕐 THỜI GIAN HỆ THỐNG (REAL-TIME)
Hiện tại: %s, %s
Năm: %d
Quý: Q%d/%d
Tháng: %d (%s)
Tuần: Tuần %d/%d
Ngày: %d (%s)

📅 KHOẢNG THỜI GIAN THAM CHIẾU:
- Hôm nay: %s
- Tuần này: %s đến %s
- Tháng này: %d đến %d (%s)
- Quý này: Q%d/%d (%s đến %s)
- Năm này: %d (%s đến %s)
- Năm ngoái: %d (%s đến %s)

🎯 GỢI Ý BIỂU THỨC:
- "3 năm gần nhất": %s
- "6 tháng tới": %s
- "quý tới": %s
- "hôm nay": %s

Lưu ý: Đây là thời gian thực của hệ thống, không phải từ dữ liệu training của model.`,
		now.Format("02/01/2006 15:04:05"),
		utils.TranslateWeekday(now.Weekday()),
		currentYear,
		quarter, currentYear,
		int(currentMonth), now.Month().String(),
		getWeekOfYear(now), currentYear,
		now.Day(),
		now.Weekday().String(),

		now.Format("02/01/2006"),
		getWeekStart(now).Format("02/01/2006"),
		getWeekEnd(now).Format("02/01/2006"),

		int(currentMonth), getLastDay(currentMonth, currentYear), currentMonth.String(),
		quarter, currentYear,
		time.Month(((quarter-1)*4)+1).String(),
		time.Month(quarter*4).String(),

		currentYear,
		"01", "12",

		currentYear-1,
		"01", "12",

		NormalizeTimeExpression("3 năm gần nhất"),
		NormalizeTimeExpression("6 tháng tới"),
		NormalizeTimeExpression("quý tới"),
		NormalizeTimeExpression("hôm nay"),
	)
}

// Helper functions
func getWeekOfYear(t time.Time) int {
	_, week := t.ISOWeek()
	return week
}

func getWeekStart(t time.Time) time.Time {
	weekday := t.Weekday()
	daysToMon := int(time.Monday - weekday)
	if daysToMon > 0 {
		daysToMon -= 7
	}
	return t.AddDate(0, 0, daysToMon)
}

func getWeekEnd(t time.Time) time.Time {
	return getWeekStart(t).AddDate(0, 0, 6)
}

func getLastDay(month time.Month, year int) int {
	days := []int{31, 30, 31, 30, 31, 31, 30, 31, 30, 31, 31, 30}
	if int(month) == 2 {
		if isLeapYear(year) {
			return 29
		}
		return 28
	}
	return days[int(month)-1]
}

func isLeapYear(year int) bool {
	if year%4 != 0 {
		return false
	}
	if year%100 != 0 {
		return true
	}
	return year%400 == 0
}