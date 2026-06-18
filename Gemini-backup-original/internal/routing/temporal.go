package routing

import (
	"strings"
	"time"

	"gemini-cli/internal/utils"
)

// TemporalIntent holds the resolved temporal information from a user query.
type TemporalIntent struct {
	Intent       string
	ResolvedDate string
	IsFuture     bool
}

// ResolveTemporal extracts temporal intent (date, future/past) from a query
// using keyword matching against Vietnamese and English time expressions.
func ResolveTemporal(query string, now time.Time) TemporalIntent {
	q := strings.ToLower(query)
	var t TemporalIntent

	// Future checks
	if utils.ContainsAny(q, "ngày mai", "ngay mai", "sắp tới", "sap toi", "tương lai", "tuong lai", "dự báo", "du bao") {
		if utils.ContainsAny(q, "gần đây", "gan day") {
			t.Intent = "latest"
		} else {
			t.IsFuture = true
		}
	}

	// Date checks
	if utils.ContainsAny(q, "hôm nay", "hom nay", "hiện tại", "hien tai") {
		t.Intent = "realtime"
		t.ResolvedDate = now.Format("2006-01-02")
	} else if utils.ContainsAny(q, "hôm qua", "hom qua") {
		t.Intent = "latest"
		yesterday := now.AddDate(0, 0, -1)
		t.ResolvedDate = yesterday.Format("2006-01-02")
	} else if utils.ContainsAny(q, "thứ hai vừa rồi", "thu hai vua roi", "thứ hai tuần này", "thu hai tuan nay") {
		t.Intent = "historical"
		daysToSubtract := int(now.Weekday()) - 1
		if daysToSubtract < 0 {
			daysToSubtract = 6
		}
		lastMonday := now.AddDate(0, 0, -daysToSubtract)
		t.ResolvedDate = lastMonday.Format("2006-01-02")
	}

	if utils.ContainsAny(q, "6 tháng đầu năm 2025", "6 thang dau nam 2025") {
		t.Intent = "historical"
		t.ResolvedDate = "2025-06-30"
	} else if utils.ContainsAny(q, "năm 2023", "nam 2023") {
		t.Intent = "historical"
		t.ResolvedDate = "2023-12-31"
	} else if utils.ContainsAny(q, "năm 2024", "nam 2024") {
		t.Intent = "historical"
		t.ResolvedDate = "2024-12-31"
	} else if utils.ContainsAny(q, "năm 2025", "nam 2025") {
		t.Intent = "historical"
	}

	if utils.ContainsAny(q, "10 năm qua", "10 nam qua") {
		t.Intent = "historical"
	} else if utils.ContainsAny(q, "3 năm gần đây", "3 nam gan day") {
		t.Intent = "historical"
	} else if utils.ContainsAny(q, "những năm gần đây", "nhung nam gan day") {
		t.Intent = "latest"
	}

	return t
}
