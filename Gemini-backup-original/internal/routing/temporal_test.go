package routing

import (
	"testing"
	"time"
)

func TestResolveTemporal(t *testing.T) {
	// Fixed reference time: Wednesday, June 11, 2025
	now := time.Date(2025, 6, 11, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name             string
		query            string
		expectedIntent   string
		expectedDate     string
		expectedFuture   bool
	}{
		// Realtime queries
		{
			name:           "hôm nay",
			query:          "giá cổ phiếu hôm nay",
			expectedIntent: "realtime",
			expectedDate:   "2025-06-11",
		},
		{
			name:           "hom nay no accent",
			query:          "giá cổ phiếu hom nay",
			expectedIntent: "realtime",
			expectedDate:   "2025-06-11",
		},
		{
			name:           "hiện tại",
			query:          "tình hình thị trường hiện tại",
			expectedIntent: "realtime",
			expectedDate:   "2025-06-11",
		},

		// Historical - yesterday
		{
			name:           "hôm qua",
			query:          "giá vàng hôm qua",
			expectedIntent: "latest",
			expectedDate:   "2025-06-10",
		},

		// Historical - specific years
		{
			name:           "năm 2023",
			query:          "báo cáo tài chính năm 2023",
			expectedIntent: "historical",
			expectedDate:   "2023-12-31",
		},
		{
			name:           "năm 2024",
			query:          "kết quả kinh doanh năm 2024",
			expectedIntent: "historical",
			expectedDate:   "2024-12-31",
		},
		{
			name:           "năm 2025",
			query:          "dữ liệu năm 2025",
			expectedIntent: "historical",
			expectedDate:   "",
		},

		// Historical - relative periods
		{
			name:           "6 tháng đầu năm 2025",
			query:          "6 tháng đầu năm 2025",
			expectedIntent: "historical",
			expectedDate:   "2025-06-30",
		},
		{
			name:           "10 năm qua",
			query:          "xu hướng 10 năm qua",
			expectedIntent: "historical",
		},
		{
			name:           "3 năm gần đây",
			query:          "tăng trưởng 3 năm gần đây",
			expectedIntent: "historical",
		},
		{
			name:           "những năm gần đây",
			query:          "những năm gần đây",
			expectedIntent: "latest",
		},

		// Future queries
		{
			name:           "ngày mai",
			query:          "dự báo ngày mai",
			expectedIntent: "",
			expectedFuture: true,
		},
		{
			name:           "tương lai",
			query:          "triển vọng tương lai",
			expectedIntent: "",
			expectedFuture: true,
		},
		{
			name:           "dự báo",
			query:          "dự báo giá cổ phiếu",
			expectedIntent: "",
			expectedFuture: true,
		},

		// Future + recent = latest
		{
			name:           "sắp tới gần đây",
			query:          "tin tức sắp tới gần đây",
			expectedIntent: "latest",
			expectedFuture: false,
		},

		// Last Monday
		{
			name:           "thứ hai vừa rồi",
			query:          "dữ liệu thứ hai vừa rồi",
			expectedIntent: "historical",
			expectedDate:   "2025-06-09",
		},

		// No temporal info
		{
			name:           "no temporal",
			query:          "phân tích cơ bản VNM",
			expectedIntent: "",
			expectedDate:   "",
			expectedFuture: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveTemporal(tt.query, now)

			if result.Intent != tt.expectedIntent {
				t.Errorf("ResolveTemporal(%q).Intent = %q, want %q", tt.query, result.Intent, tt.expectedIntent)
			}
			if result.ResolvedDate != tt.expectedDate {
				t.Errorf("ResolveTemporal(%q).ResolvedDate = %q, want %q", tt.query, result.ResolvedDate, tt.expectedDate)
			}
			if result.IsFuture != tt.expectedFuture {
				t.Errorf("ResolveTemporal(%q).IsFuture = %v, want %v", tt.query, result.IsFuture, tt.expectedFuture)
			}
		})
	}
}

func TestResolveTemporal_DefaultNow(t *testing.T) {
	result := ResolveTemporal("hôm nay", time.Now())
	if result.Intent != "realtime" {
		t.Errorf("expected 'realtime' intent for 'hôm nay', got %q", result.Intent)
	}
	if result.ResolvedDate == "" {
		t.Error("expected non-empty date for 'hôm nay'")
	}
}

func TestResolveTemporal_EmptyQuery(t *testing.T) {
	now := time.Date(2025, 6, 11, 12, 0, 0, 0, time.UTC)
	result := ResolveTemporal("", now)
	if result.Intent != "" {
		t.Errorf("expected empty intent for empty query, got %q", result.Intent)
	}
	if result.IsFuture {
		t.Error("expected IsFuture=false for empty query")
	}
}

func TestTemporalIntent_Struct(t *testing.T) {
	ti := TemporalIntent{
		Intent:       "historical",
		ResolvedDate: "2025-06-11",
		IsFuture:     false,
	}
	if ti.Intent != "historical" {
		t.Error("Intent field mismatch")
	}
	if ti.ResolvedDate != "2025-06-11" {
		t.Error("ResolvedDate field mismatch")
	}
	if ti.IsFuture != false {
		t.Error("IsFuture field mismatch")
	}
}
