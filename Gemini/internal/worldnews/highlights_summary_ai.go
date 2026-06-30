package worldnews

import (
	"fmt"
	"regexp"
	"strings"

	"gemini-cli/internal/utils"
)

const highlightSummarySystemPrompt = `Bạn là biên tập viên tài chính chuyên nghiệp của INDEXIUM.
Nhiệm vụ: viết đoạn tóm tắt bản tin sáng ngắn gọn, chính xác, bằng tiếng Việt.
Chỉ dùng dữ liệu được cung cấp — không bịa số liệu hay sự kiện.
Không dùng bullet, markdown, hoặc lời dẫn ngoài đoạn văn.`

var markdownNoise = regexp.MustCompile(`(?m)^#{1,6}\s+`)

func (s *Service) resolveHighlightSummary(
	sp, nd, wti, brent, gold, dxy *quoteSnapshot,
	news []rssItem,
	quoteLabel, digestWindow string,
) string {
	fallback := buildHighlightSummary(sp, nd, wti, brent, gold, dxy, news, quoteLabel, digestWindow)

	s.mu.RLock()
	gen := s.textGen
	s.mu.RUnlock()
	if gen == nil {
		return fallback
	}

	aiSummary, err := generateHighlightSummaryAI(
		gen,
		sp, nd, wti, brent, gold, dxy,
		news,
		quoteLabel,
		digestWindow,
	)
	if err != nil || strings.TrimSpace(aiSummary) == "" {
		fmt.Printf("⚠️ [WorldNews] AI highlight summary failed, using template: %v\n", err)
		return fallback
	}
	return aiSummary
}

func generateHighlightSummaryAI(
	gen TextGenerator,
	sp, nd, wti, brent, gold, dxy *quoteSnapshot,
	news []rssItem,
	quoteLabel, digestWindow string,
) (string, error) {
	if gen == nil {
		return "", fmt.Errorf("text generator is nil")
	}

	userPrompt := utils.RenderPromptTemplate("world_news_highlight_summary.txt", map[string]string{
		"DIGEST_WINDOW": digestWindow,
		"QUOTE_LABEL":   quoteLabel,
		"MARKET_DATA":   buildHighlightSummaryMarketData(sp, nd, wti, brent, gold, dxy, quoteLabel),
		"NEWS_ITEMS":    buildHighlightSummaryNewsData(news),
	})
	if strings.TrimSpace(userPrompt) == "" {
		return "", fmt.Errorf("empty prompt template")
	}

	raw, err := gen.GenerateText(highlightSummarySystemPrompt, userPrompt)
	if err != nil {
		return "", err
	}

	cleaned := sanitizeAISummary(raw)
	if cleaned == "" {
		return "", fmt.Errorf("empty AI response")
	}
	return fitSummaryLength(cleaned, HighlightSummaryTargetRunes), nil
}

func buildHighlightSummaryMarketData(
	sp, nd, wti, brent, gold, dxy *quoteSnapshot,
	quoteLabel string,
) string {
	var lines []string

	if sp != nil {
		dir := "giảm"
		if sp.IsPositive {
			dir = "tăng"
		}
		_, pct, _ := formatChange(sp.Change, sp.ChangePct)
		line := fmt.Sprintf(
			"- S&P 500 %s %s, chốt phiên %s ở %s",
			dir, pct, quoteLabel, formatPrice(sp.Symbol, sp.Price),
		)
		if nd != nil {
			_, nPct, _ := formatChange(nd.Change, nd.ChangePct)
			line += fmt.Sprintf("; Nasdaq Composite %s, đóng cửa %s", nPct, formatPrice(nd.Symbol, nd.Price))
		}
		lines = append(lines, line)
	}

	if wti != nil {
		_, pct, _ := formatChange(wti.Change, wti.ChangePct)
		lines = append(lines, fmt.Sprintf("- Dầu WTI %s (%s)", pct, formatPrice(wti.Symbol, wti.Price)))
	}
	if brent != nil {
		_, pct, _ := formatChange(brent.Change, brent.ChangePct)
		lines = append(lines, fmt.Sprintf("- Dầu Brent %s (%s)", pct, formatPrice(brent.Symbol, brent.Price)))
	}
	if gold != nil {
		_, pct, _ := formatChange(gold.Change, gold.ChangePct)
		lines = append(lines, fmt.Sprintf("- Vàng giao ngay %s (%s)", pct, formatPrice(gold.Symbol, gold.Price)))
	}
	if dxy != nil {
		_, pct, _ := formatChange(dxy.Change, dxy.ChangePct)
		lines = append(lines, fmt.Sprintf("- Chỉ số USD (DXY) %s (%s)", pct, formatPrice(dxy.Symbol, dxy.Price)))
	}

	if len(lines) == 0 {
		return "(không có dữ liệu thị trường)"
	}
	return strings.Join(lines, "\n")
}

func buildHighlightSummaryNewsData(news []rssItem) string {
	if len(news) == 0 {
		return "(không có tin nổi bật trong khung thời gian)"
	}

	var lines []string
	for i, it := range news {
		if i >= 8 {
			break
		}
		when := it.PubDate.In(vnTimezone).Format("02/01/2006 15:04")
		source := strings.TrimSpace(it.Source)
		if source == "" {
			source = "Nguồn tin"
		}
		title := strings.TrimSpace(it.Title)
		if title == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("- %s (%s): %s", source, when, title))
	}

	if len(lines) == 0 {
		return "(không có tin nổi bật trong khung thời gian)"
	}
	return strings.Join(lines, "\n")
}

func sanitizeAISummary(text string) string {
	text = strings.TrimSpace(text)
	text = strings.Trim(text, `"'`)
	text = markdownNoise.ReplaceAllString(text, "")
	text = strings.ReplaceAll(text, "**", "")
	text = strings.ReplaceAll(text, "__", "")
	text = strings.ReplaceAll(text, "\n", " ")
	return strings.Join(strings.Fields(text), " ")
}