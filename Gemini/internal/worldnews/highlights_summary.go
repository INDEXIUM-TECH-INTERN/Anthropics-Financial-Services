package worldnews

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	HighlightSummaryMinWords = 800
	HighlightSummaryMaxWords = 1000
)

func buildHighlightSummary(
	sp, nd, wti, brent, gold, dxy *quoteSnapshot,
	news []rssItem,
	quoteLabel, digestWindow string,
) string {
	var sentences []string

	sentences = append(sentences, fmt.Sprintf(
		"Bản tin sáng tổng hợp diễn biến tài chính toàn cầu trước 07:00 (GMT+7), khung %s.",
		digestWindow,
	))

	if sp != nil {
		dir := "giảm điểm"
		if sp.IsPositive {
			dir = "tăng điểm"
		}
		_, pct, _ := formatChange(sp.Change, sp.ChangePct)
		market := fmt.Sprintf(
			"Thị trường chứng khoán Mỹ: S&P 500 %s %s, chốt phiên %s ở %s",
			dir, pct, quoteLabel, formatPrice(sp.Symbol, sp.Price),
		)
		if nd != nil {
			_, nPct, _ := formatChange(nd.Change, nd.ChangePct)
			market += fmt.Sprintf("; Nasdaq Composite %s, đóng cửa %s", nPct, formatPrice(nd.Symbol, nd.Price))
		}
		sentences = append(sentences, market+".")
	}

	var commodityParts []string
	if wti != nil && brent != nil {
		_, wPct, _ := formatChange(wti.Change, wti.ChangePct)
		_, bPct, _ := formatChange(brent.Change, brent.ChangePct)
		commodityParts = append(commodityParts, fmt.Sprintf(
			"dầu WTI %s (%s) và Brent %s (%s)",
			wPct, formatPrice(wti.Symbol, wti.Price),
			bPct, formatPrice(brent.Symbol, brent.Price),
		))
	}
	if gold != nil {
		_, gPct, _ := formatChange(gold.Change, gold.ChangePct)
		commodityParts = append(commodityParts, fmt.Sprintf(
			"vàng giao ngay %s (%s)",
			gPct, formatPrice(gold.Symbol, gold.Price),
		))
	}
	if dxy != nil {
		_, dPct, _ := formatChange(dxy.Change, dxy.ChangePct)
		commodityParts = append(commodityParts, fmt.Sprintf("chỉ số USD (DXY) %s", dPct))
	}
	if len(commodityParts) > 0 {
		sentences = append(sentences, "Nhóm hàng hóa và ngoại tệ: "+strings.Join(commodityParts, ", ")+".")
	}

	if len(news) > 0 {
		sentences = append(sentences, "Trên mặt trận tin tức, các dòng sự kiện đáng chú ý gồm:")
		for i, it := range news {
			if i >= 10 {
				break
			}
			when := it.PubDate.In(vnTimezone).Format("02/01/2006 15:04")
			source := it.Source
			if source == "" {
				source = "Nguồn tin"
			}
			sentences = append(sentences, fmt.Sprintf(
				"%s (%s) — %s; góc nhìn thị trường xoay quanh %s",
				source, when, it.Title, summarizeNewsAngle(it.Title),
			)+".")
		}
	}

	sentences = append(sentences, "Nhà đầu tư nên theo dõi sát diễn biến lãi suất, chính sách tiền tệ và các báo cáo kinh tế vĩ mô trong phiên giao dịch sắp tới để điều chỉnh danh mục phù hợp.")

	text := strings.Join(sentences, " ")
	return fitSummaryWordLength(text, HighlightSummaryMinWords, HighlightSummaryMaxWords)
}

func wordCount(text string) int {
	return len(strings.Fields(strings.TrimSpace(text)))
}

func summarizeNewsAngle(title string) string {
	lower := strings.ToLower(title)
	switch {
	case strings.Contains(lower, "fed"), strings.Contains(lower, "rate"), strings.Contains(lower, "lãi suất"):
		return "xu hướng lãi suất và định hướng chính sách Fed"
	case strings.Contains(lower, "oil"), strings.Contains(lower, "dầu"), strings.Contains(lower, "opec"):
		return "cung cầu năng lượng và biến động giá dầu"
	case strings.Contains(lower, "gold"), strings.Contains(lower, "vàng"):
		return "nhu cầu trú ẩn an toàn và giá vàng"
	case strings.Contains(lower, "china"), strings.Contains(lower, "trung quốc"):
		return "triển vọng kinh tế Trung Quốc và tác động lan tỏa"
	case strings.Contains(lower, "earnings"), strings.Contains(lower, "profit"):
		return "kết quả kinh doanh doanh nghiệp và kỳ vọng lợi nhuận"
	default:
		return "tác động lên tâm lý thị trường tài chính toàn cầu"
	}
}

func fitSummaryWordLength(text string, minWords, maxWords int) string {
	text = strings.Join(strings.Fields(text), " ")
	if wordCount(text) > maxWords {
		return trimToMaxWords(text, maxWords)
	}
	if wordCount(text) < minWords {
		return padSummaryWords(text, minWords, maxWords)
	}
	return ensureSentenceEnding(text, maxWords*12)
}

// normalizeAISummaryWords trims AI prose to the word budget without "..." padding.
func normalizeAISummaryWords(text string, minWords, maxWords int) string {
	text = strings.Join(strings.Fields(text), " ")
	text = stripTrailingEllipsis(text)
	if text == "" {
		return ""
	}
	if wordCount(text) > maxWords {
		if trimmed := trimToMaxWordsAtSentence(text, maxWords); trimmed != "" {
			return trimmed
		}
		return trimToMaxWords(text, maxWords)
	}
	return ensureSentenceEnding(text, maxWords*12)
}

func trimToMaxWords(text string, maxWords int) string {
	words := strings.Fields(text)
	if len(words) <= maxWords {
		return text
	}
	return strings.Join(words[:maxWords], " ")
}

func trimToMaxWordsAtSentence(text string, maxWords int) string {
	words := strings.Fields(text)
	if len(words) <= maxWords {
		return ensureSentenceEnding(text, maxWords*12)
	}
	chunk := strings.Join(words[:maxWords], " ")
	best := -1
	for _, sep := range []string{". ", "! ", "? "} {
		if idx := strings.LastIndex(chunk, sep); idx > best {
			best = idx
		}
	}
	if best > 0 {
		return strings.TrimSpace(chunk[:best+1])
	}
	return trimToMaxWords(text, maxWords)
}

func padSummaryWords(text string, minWords, maxWords int) string {
	padding := " Diễn biến trên phản ánh sự phân hóa giữa kỳ vọng tăng trưởng, rủi ro địa chính trị, chu kỳ chính sách tiền tệ và tâm lý nhà đầu tư toàn cầu, đòi hỏi duy trì kỷ luật quản trị rủi ro và theo dõi sát các chỉ báo vĩ mô trong các phiên giao dịch sắp tới."
	for wordCount(text) < minWords {
		text += padding
		if wordCount(text) > maxWords {
			return trimToMaxWords(text, maxWords)
		}
	}
	return ensureSentenceEnding(text, maxWords*12)
}

func stripTrailingEllipsis(text string) string {
	text = strings.TrimSpace(text)
	for {
		changed := false
		if strings.HasSuffix(text, "…") {
			text = strings.TrimSpace(strings.TrimSuffix(text, "…"))
			changed = true
		}
		if strings.HasSuffix(text, "...") {
			text = strings.TrimSpace(strings.TrimSuffix(text, "..."))
			changed = true
		}
		if strings.HasSuffix(text, "..") {
			text = strings.TrimSpace(strings.TrimSuffix(text, ".."))
			changed = true
		}
		if !changed {
			break
		}
	}
	return strings.TrimRight(text, ".")
}

func ensureSentenceEnding(text string, max int) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if strings.HasSuffix(text, ".") || strings.HasSuffix(text, "!") || strings.HasSuffix(text, "?") {
		return text
	}
	if utf8.RuneCountInString(text)+1 <= max {
		return text + "."
	}
	return text
}

