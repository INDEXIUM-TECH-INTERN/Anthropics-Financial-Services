package routing

import "strings"

// IsCasualGreeting returns true when the input looks like a short social
// greeting (e.g. "hi", "hello", "xin chao"). Only matches known greeting
// patterns — does NOT use a length-based catch-all, so short queries like
// stock tickers ("VNM", "BTC") are not misclassified as greetings.
func IsCasualGreeting(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	lower = strings.NewReplacer(".", "", "!", "", "?", "", ",", "").Replace(lower)
	lower = RemoveAccents(lower)

	// Known greeting phrases (Vietnamese + English)
	greetings := []string{
		"hi", "hello", "hey", "alo",
		"xin chao", "chao ban", "chao",
		"ten ban la gi", "ban la ai", "ai do", "who are you",
	}

	for _, g := range greetings {
		if lower == g || strings.HasPrefix(lower, g+" ") {
			return true
		}
	}

	// Identity/introduction questions
	if strings.Contains(lower, "la ai") || strings.Contains(lower, "ten gi") {
		return true
	}

	return false
}

// RemoveAccents converts Vietnamese accented characters to their plain ASCII
// equivalents so that greeting detection works regardless of accent marks.
func RemoveAccents(s string) string {
	accents := map[string]string{
		"a": "áàảãạăắằẳẵặâấầẩẫậ",
		"d": "đ",
		"e": "éèẻẽẹêếềểễệ",
		"i": "íìỉĩị",
		"o": "óòỏõọôốồổỗộơớờởỡợ",
		"u": "úùủũụưứừửữự",
		"y": "ýỳỷỹỵ",
	}
	for unaccented, accentedChars := range accents {
		for _, char := range accentedChars {
			s = strings.ReplaceAll(s, string(char), unaccented)
		}
	}
	return s
}
