package routing

import "strings"

// IsCasualGreeting returns true when the input looks like a short social
// greeting (e.g. "hi", "hello", "xin chao").
func IsCasualGreeting(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	lower = strings.NewReplacer(".", "", "!", "", "?", "", ",", "").Replace(lower)
	lower = RemoveAccents(lower)

	greetings := []string{
		"hi", "hello", "xin chao", "chao ban", "chao", "hey", "alo",
		"ten ban la gi", "ban la ai", "ai do", "who are you",
		"giup toi", "huong dan", "su dung", "test",
	}

	for _, g := range greetings {
		if lower == g || strings.HasPrefix(lower, g+" ") || strings.Contains(lower, "la ai") || strings.Contains(lower, "ten gi") {
			return true
		}
	}
	return len(lower) < 5
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
