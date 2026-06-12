package routing

import (
	"testing"
)

func TestIsCasualGreeting(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Basic English greetings
		{"hello", "hello", true},
		{"hi", "hi", true},
		{"hey", "hey", true},
		{"alo", "alo", true},

		// Vietnamese greetings
		{"xin chao", "xin chao", true},
		{"chao ban", "chao ban", true},
		{"chao", "chao", true},

		// Identity questions
		{"ten ban la gi", "ten ban la gi", true},
		{"ban la ai", "ban la ai", true},
		{"ai do", "ai do", true},
		{"who are you", "who are you", true},

		// Help requests
		{"giup toi", "giup toi", true},
		{"huong dan", "huong dan", true},
		{"su dung", "su dung", true},
		{"test", "test", true},

		// With punctuation
		{"hello with exclamation", "hello!", true},
		{"hi with period", "hi.", true},
		{"hello with question", "hello?", true},

		// With spaces
		{"hello with trailing space", "hello ", true},
		{"hi with leading space", " hi", true},

		// Short inputs (< 5 chars)
		{"short abc", "abc", true},
		{"short x", "x", true},
		{"empty string", "", true},

		// Non-greetings (longer, specific queries)
		{"financial query", "phân tích báo cáo tài chính Vinamilk 2025", false},
		{"stock query", "giá cổ phiếu VNM hôm nay", false},
		{"earnings query", "kết quả kinh doanh quý 3", false},
		{"model query", "xây dựng mô hình DCF cho FPT", false},

		// Contains "la ai" substring
		{"contains la ai", "đây la ai", true},
		{"contains ten gi", "cai nay ten gi", true},

		// Accented Vietnamese
		{"xin chào with accents", "xin chào", true},
		{"chào bạn with accents", "chào bạn", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCasualGreeting(tt.input)
			if result != tt.expected {
				t.Errorf("IsCasualGreeting(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRemoveAccents(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"no accents", "hello", "hello"},
		{"a accents", "áàảãạăắằẳẵặâấầẩẫậ", "aaaaaaaaaaaaaaaaa"},
		{"d accent", "đ", "d"},
		{"e accents", "éèẻẽẹêếềểễệ", "eeeeeeeeeee"},
		{"i accents", "íìỉĩị", "iiiii"},
		{"o accents", "óòỏõọôốồổỗộơớờởỡợ", "ooooooooooooooooo"},
		{"u accents", "úùủũụưứừửữự", "uuuuuuuuuuu"},
		{"y accents", "ýỳỷỹỵ", "yyyyy"},
		{"mixed", "Tiếng Việt", "Tieng Viet"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RemoveAccents(tt.input)
			if result != tt.expected {
				t.Errorf("RemoveAccents(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
