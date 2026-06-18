package tools

import (
	"fmt"
	"strconv"
	"strings"

	"gemini-cli/internal/tools/market"
	"gemini-cli/internal/tools/registry"
	"gemini-cli/internal/tools/scraper"
)

type MarketQueryPlan = market.MarketQueryPlan
type LoadedDocument = registry.LoadedDocument

func SearchTavily(query string) string {
	return market.SearchTavily(query)
}

func SearchGoogle(query string) string {
	return market.SearchGoogle(query)
}

func ScrapeWeb(targetURL string) string {
	return scraper.ScrapeWeb(targetURL)
}

func Calculate(expression string) string {
	fmt.Printf("🧮 [Tool] Đang tính toán: %s...\n", expression)
	result, err := evalExpression(expression)
	if err != nil {
		return fmt.Sprintf("Lỗi tính toán: %v", err)
	}
	return fmt.Sprintf("Kết quả của '%s' = %v", expression, result)
}

// evalExpression đánh giá biểu thức toán học cơ bản (+, -, *, /, ^, %, parentheses)
func evalExpression(expr string) (float64, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return 0, fmt.Errorf("biểu thức rỗng")
	}
	parser := &exprParser{s: expr, pos: 0}
	return parser.parse()
}

// exprParser implements a simple recursive descent parser for math expressions
type exprParser struct {
	s   string
	pos int
}

func (p *exprParser) parse() (float64, error) {
	return p.parseAddSub()
}

func (p *exprParser) parseAddSub() (float64, error) {
	left, err := p.parseMulDiv()
	if err != nil {
		return 0, err
	}
	for p.pos < len(p.s) {
		op := p.s[p.pos]
		if op != '+' && op != '-' {
			break
		}
		p.pos++
		right, err := p.parseMulDiv()
		if err != nil {
			return 0, err
		}
		if op == '+' {
			left += right
		} else {
			left -= right
		}
	}
	return left, nil
}

func (p *exprParser) parseMulDiv() (float64, error) {
	left, err := p.parsePower()
	if err != nil {
		return 0, err
	}
	for p.pos < len(p.s) {
		op := p.s[p.pos]
		if op != '*' && op != '/' && op != '%' {
			break
		}
		p.pos++
		right, err := p.parsePower()
		if err != nil {
			return 0, err
		}
		switch op {
		case '*':
			left *= right
		case '/':
			if right == 0 {
				return 0, fmt.Errorf("chia cho 0")
			}
			left /= right
		case '%':
			left = float64(int64(left) % int64(right))
		}
	}
	return left, nil
}

func (p *exprParser) parsePower() (float64, error) {
	base, err := p.parseUnary()
	if err != nil {
		return 0, err
	}
	if p.pos < len(p.s) && p.s[p.pos] == '^' {
		p.pos++
		exp, err := p.parseUnary()
		if err != nil {
			return 0, err
		}
		result := 1.0
		for i := 0; i < int(exp); i++ {
			result *= base
		}
		return result, nil
	}
	return base, nil
}

func (p *exprParser) parseUnary() (float64, error) {
	if p.pos < len(p.s) && p.s[p.pos] == '-' {
		p.pos++
		val, err := p.parsePrimary()
		if err != nil {
			return 0, err
		}
		return -val, nil
	}
	if p.pos < len(p.s) && p.s[p.pos] == '+' {
		p.pos++
	}
	return p.parsePrimary()
}

func (p *exprParser) parsePrimary() (float64, error) {
	for p.pos < len(p.s) && p.s[p.pos] == ' ' {
		p.pos++
	}
	if p.pos >= len(p.s) {
		return 0, fmt.Errorf("biểu thức không hợp lệ")
	}
	if p.s[p.pos] == '(' {
		p.pos++
		val, err := p.parseAddSub()
		if err != nil {
			return 0, err
		}
		if p.pos >= len(p.s) || p.s[p.pos] != ')' {
			return 0, fmt.Errorf("thiếu dấu đóng ngoặc")
		}
		p.pos++
		return val, nil
	}
	start := p.pos
	for p.pos < len(p.s) && ((p.s[p.pos] >= '0' && p.s[p.pos] <= '9') || p.s[p.pos] == '.' || p.s[p.pos] == ',') {
		p.pos++
	}
	if start == p.pos {
		return 0, fmt.Errorf("ký tự không hợp lệ: '%c'", p.s[p.pos])
	}
	numStr := strings.ReplaceAll(p.s[start:p.pos], ",", "")
	return strconv.ParseFloat(numStr, 64)
}

func NeedsRealtimeData(query string) bool {
	return market.NeedsRealtimeData(query)
}

func BuildMarketQueryPlan(query string) MarketQueryPlan {
	return market.BuildMarketQueryPlan(query)
}

func LoadDocument(docType, name string) string {
	return registry.LoadDocument(docType, name)
}

func LoadDocumentWithMetadata(docType, name string) LoadedDocument {
	return registry.LoadDocumentWithMetadata(docType, name)
}

func GetRoutingGuide() string {
	return registry.GetRoutingGuide()
}

// Time validation utilities - placeholder for future implementation
func ValidateAndFixTimeRanges(text string) string {
	// TODO: Implement time validation using new time parser
	return NormalizeTimeExpression(text)
}

func GetCurrentTimeInfo() string {
	return GetCurrentTimeInfoEx()
}

// NormalizeTimeExpression chuẩn hóa và giải thích biểu thức thời gian
// NormalizeTimeExpression chuẩn hóa và giải thích biểu thức thời gian
