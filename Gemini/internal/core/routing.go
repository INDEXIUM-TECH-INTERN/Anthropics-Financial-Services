package core

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"gemini-cli/internal/api"
	"gemini-cli/internal/tools"
	"gemini-cli/internal/utils"
)

type RoutePlan struct {
	Agent    string   `json:"agent"`
	Skills   []string `json:"skills"`
	Temporal struct {
		Intent       string `json:"intent"`        // e.g., "historical", "latest", "realtime"
		ResolvedDate string `json:"resolved_date"` // Định dạng YYYY-MM-DD
		IsFuture     bool   `json:"is_future"`
	} `json:"temporal"`
	Reason string `json:"reason"`
}

var allowedAgents = map[string]bool{
	"pitch-agent":        true,
	"meeting-prep-agent": true,
	"market-researcher":  true,
	"earnings-reviewer":  true,
	"model-builder":      true,
	"valuation-reviewer": true,
	"gl-reconciler":      true,
	"month-end-closer":   true,
	"statement-auditor":  true,
	"kyc-screener":       true,
}

var allowedSkillsByAgent = map[string]map[string]bool{
	"pitch-agent": {
		"pitch-deck":             true,
		"datapack-builder":       true,
		"cim-builder":            true,
		"teaser":                 true,
		"buyer-list":             true,
		"comps-analysis":         true,
		"precedent-transactions": true,
		"lbo-model":              true,
		"merger-model":           true,
	},
	"meeting-prep-agent": {
		"briefing-pack":       true,
		"biography-generator": true,
		"company-profile":     true,
		"news-digest":         true,
	},
	"market-researcher": {
		"sector-overview":      true,
		"competitive-analysis": true,
		"comps-analysis":       true,
		"idea-generation":      true,
		"thesis-tracker":       true,
		"catalyst-calendar":    true,
	},
	"earnings-reviewer": {
		"earnings-analysis":   true,
		"earnings-preview":    true,
		"initiating-coverage": true,
		"model-update":        true,
		"morning-note":        true,
		"xlsx-author":         true,
	},
	"model-builder": {
		"dcf-model":         true,
		"lbo-model":         true,
		"3-statement-model": true,
		"merger-model":      true,
		"xlsx-author":       true,
		"audit-xls":         true,
	},
	"valuation-reviewer": {
		"valuation-review": true,
		"gp-reporting":     true,
		"lp-reporting":     true,
	},
	"gl-reconciler": {
		"break-detection":     true,
		"root-cause-analysis": true,
		"sign-off-routing":    true,
	},
	"month-end-closer": {
		"accruals":            true,
		"roll-forwards":       true,
		"variance-commentary": true,
	},
	"statement-auditor": {
		"lp-statement-audit":        true,
		"distribution-verification": true,
	},
	"kyc-screener": {
		"onboarding-doc-parsing": true,
		"gap-flagging":           true,
	},
}

// SelectRoutePlan routes a query to the best agent using a lightweight heuristic only.
// It does NOT create a full Agent (no provider initialization) — safe for testing.
func SelectRoutePlan(query string) RoutePlan {
	now := time.Now()
	if dateOverride := os.Getenv("SYSTEM_DATE_OVERRIDE"); dateOverride != "" {
		if t, err := time.Parse("2006-01-02", dateOverride); err == nil {
			now = t
		}
	}
	return heuristicRoutePlan(query, now)
}

func (a *Agent) selectRoutePlan() RoutePlan {
	api.BroadcastLog("Đang phân tích yêu cầu để chọn Agent tối ưu...", "process")
	routerSystemPrompt := utils.LoadPrompt("router_system_prompt.txt")

	now := time.Now()
	if dateOverride := os.Getenv("SYSTEM_DATE_OVERRIDE"); dateOverride != "" {
		if t, err := time.Parse("2006-01-02", dateOverride); err == nil {
			now = t
		}
	}

	routerUserPrompt := buildRouterUserPrompt(a.userInput, tools.GetRoutingGuide(), now)

	raw, err := a.routeWithProviderFallback(routerSystemPrompt, routerUserPrompt)
	var route RoutePlan
	if err != nil {
		fmt.Printf("⚠️ [Router] Provider error: %v. Falling back to heuristic router.\n", err)
		route = heuristicRoutePlan(a.userInput, now)
	} else {
		fmt.Printf("\n[Router] Raw AI Response (first 200 chars): %.200s\n", raw)
		var parseErr error
		route, parseErr = parseRoutePlan(raw)
		if parseErr != nil {
			fmt.Printf("⚠️ [Router] Parse error: %v. Falling back to heuristic router.\n", parseErr)
			route = heuristicRoutePlan(a.userInput, now)
		}
	}

	api.BroadcastEvent(
		fmt.Sprintf("Đã chọn Agent: %s (Lý do: %s)", route.Agent, route.Reason),
		"agent_selected",
		map[string]interface{}{
			"agent":  route.Agent,
			"reason": route.Reason,
			"skills": route.Skills,
		},
	)
	return sanitizeRoutePlan(route, a.userInput)
}

func containsAny(s string, keywords ...string) bool {
	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}

func heuristicRoutePlan(userInput string, now time.Time) RoutePlan {
	q := strings.ToLower(userInput)
	var route RoutePlan
	route.Agent = "market-researcher"
	
	// Determine Agent
	if containsAny(q, "ban lãnh đạo", "lãnh đạo", "ban lanh dao", "board of directors", "leadership", "executive", "ban điều hành", "ban dieu hanh", "hội đồng quản trị", "hoi dong quan tri") {
		route.Agent = "meeting-prep-agent"
	} else if containsAny(q, "nằm ở trang nào", "nam o trang nao", "trang mấy", "trang may", "audit", "kiểm toán", "kiem toan") {
		route.Agent = "statement-auditor"
	} else if containsAny(q, "báo cáo thu nhập", "bao cao thu nhap", "trích dẫn báo cáo", "trich dan bao cao", "6 tháng đầu năm", "6 thang dau nam", "quý", "quy", "doanh thu", "lợi nhuận") {
		route.Agent = "earnings-reviewer"
	} else if containsAny(q, "10 năm qua", "10 nam qua", "dự phóng", "du phong", "dcf-model", "định giá", "valuation", "lbo-model") {
		route.Agent = "model-builder"
	} else if containsAny(q, "so sánh", "so sanh", "phân tích kỹ thuật", "phan tich ky thuat", "giá cổ phiếu", "gia co phieu", "lợi thế gì", "loi the gi", "ngành ngân hàng", "nganh ngan hang") {
		route.Agent = "market-researcher"
	} else if containsAny(q, "báo cáo tài chính năm 2024", "báo cáo tài chính năm 2023", "báo cáo năm") {
		route.Agent = "earnings-reviewer"
	}

	// 2. Determine Temporal Intent & Date
	route.Temporal.Intent = ""
	route.Temporal.ResolvedDate = ""
	route.Temporal.IsFuture = false

	// Future checks
	if containsAny(q, "ngày mai", "ngay mai", "sắp tới", "sap toi", "tương lai", "tuong lai", "dự báo", "du bao") {
		if containsAny(q, "gần đây", "gan day") {
			route.Temporal.Intent = "latest"
		} else {
			route.Temporal.IsFuture = true
		}
	}

	// Date checks
	if containsAny(q, "hôm nay", "hom nay", "hiện tại", "hien tai") {
		route.Temporal.Intent = "realtime"
		route.Temporal.ResolvedDate = now.Format("2006-01-02")
	} else if containsAny(q, "hôm qua", "hom qua") {
		route.Temporal.Intent = "latest"
		yesterday := now.AddDate(0, 0, -1)
		route.Temporal.ResolvedDate = yesterday.Format("2006-01-02")
	} else if containsAny(q, "thứ hai vừa rồi", "thu hai vua roi", "thứ hai tuần này", "thu hai tuan nay") {
		route.Temporal.Intent = "historical"
		daysToSubtract := int(now.Weekday()) - 1
		if daysToSubtract < 0 {
			daysToSubtract = 6
		}
		lastMonday := now.AddDate(0, 0, -daysToSubtract)
		route.Temporal.ResolvedDate = lastMonday.Format("2006-01-02")
	}

	if containsAny(q, "6 tháng đầu năm 2025", "6 thang dau nam 2025") {
		route.Temporal.Intent = "historical"
		route.Temporal.ResolvedDate = "2025-06-30"
	} else if containsAny(q, "năm 2023", "nam 2023") {
		route.Temporal.Intent = "historical"
		route.Temporal.ResolvedDate = "2023-12-31"
	} else if containsAny(q, "năm 2024", "nam 2024") {
		route.Temporal.Intent = "historical"
		route.Temporal.ResolvedDate = "2024-12-31"
	} else if containsAny(q, "năm 2025", "nam 2025") {
		route.Temporal.Intent = "historical"
	}

	if containsAny(q, "10 năm qua", "10 nam qua") {
		route.Temporal.Intent = "historical"
	} else if containsAny(q, "3 năm gần đây", "3 nam gan day") {
		route.Temporal.Intent = "historical"
	} else if containsAny(q, "những năm gần đây", "nhung nam gan day") {
		route.Temporal.Intent = "latest"
	}

	route.Skills = guessSkillsForAgent(route.Agent)
	route.Reason = "Heuristically determined based on query patterns."
	return route
}

func guessSkillsForAgent(agent string) []string {
	switch agent {
	case "pitch-agent":
		return []string{"pitch-deck"}
	case "meeting-prep-agent":
		return []string{"briefing-pack"}
	case "market-researcher":
		return []string{"sector-overview"}
	case "earnings-reviewer":
		return []string{"earnings-analysis"}
	case "model-builder":
		return []string{"dcf-model"}
	case "valuation-reviewer":
		return []string{"valuation-review"}
	case "gl-reconciler":
		return []string{"break-detection"}
	case "month-end-closer":
		return []string{"accruals"}
	case "statement-auditor":
		return []string{"lp-statement-audit"}
	case "kyc-screener":
		return []string{"onboarding-doc-parsing"}
	default:
		return []string{"sector-overview"}
	}
}


func buildRouterUserPrompt(userInput, routingGuide string, now time.Time) string {
	return utils.RenderPromptTemplate("router_user_prompt.txt", map[string]string{
		"SYSTEM_TIME":    now.Format("02/01/2006 15:04:05"),
		"SYSTEM_WEEKDAY": utils.TranslateWeekday(now.Weekday()),
		"USER_REQUEST":   userInput,
		"ROUTING_GUIDE":  extractRoutingGuideSummary(routingGuide),
		"ROUTER_CATALOG": utils.LoadPrompt("router_catalog.txt"),
	})
}

func (a *Agent) routeWithProviderFallback(systemPrompt, userPrompt string) (string, error) {
	return a.GetProvider().GenerateText(systemPrompt, userPrompt)
}

func fallbackRoutePlan() RoutePlan {
	return RoutePlan{
		Agent:  "market-researcher",
		Skills: []string{"sector-overview"},
		Reason: "Default fallback for general requests.",
	}
}

func sanitizeRoutePlan(route RoutePlan, userInput string) RoutePlan {
	// Bắt buộc phải nằm trong danh sách cho phép (tránh hallucination như "administrative-researcher")
	if !allowedAgents[route.Agent] {
		fmt.Printf("⚠️ [Router] Agent '%s' không nằm trong danh sách allowedAgents. Fallback an toàn.\n", route.Agent)
		return fallbackRoutePlan()
	}

	// Kiểm tra xem agent có tồn tại trong repo không bằng cách thử nạp metadata
	agentDoc := tools.LoadDocumentWithMetadata("agent", route.Agent)
	if strings.HasPrefix(agentDoc.Content, "Lỗi:") {
		fmt.Printf("⚠️ [Router] Agent '%s' không tồn tại trong repo. Fallback.\n", route.Agent)
		return fallbackRoutePlan()
	}

	// Lọc skills dựa trên thực tế có trong repo
	var validSkills []string
	for _, skill := range route.Skills {
		skillDoc := tools.LoadDocumentWithMetadata("skill", route.Agent+"/"+skill)
		if !strings.HasPrefix(skillDoc.Content, "Lỗi:") {
			validSkills = append(validSkills, skill)
		} else {
			fmt.Printf("⚠️ [Router] Skill '%s' không tồn tại cho agent '%s'. Bỏ qua.\n", skill, route.Agent)
		}
	}

	if len(validSkills) == 0 {
		route.Skills = guessSkillsForAgent(route.Agent)
	} else {
		route.Skills = validSkills
	}

	return route
}

func parseRoutePlan(raw string) (RoutePlan, error) {
	// Tìm JSON trong chuỗi văn bản
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start == -1 || end == -1 || end < start {
		return RoutePlan{}, fmt.Errorf("no JSON found")
	}

	jsonStr := raw[start : end+1]

	// Loại bỏ các comment dòng (// ...) thường thấy trong output AI
	var cleanLines []string
	for _, line := range strings.Split(jsonStr, "\n") {
		if idx := strings.Index(line, "//"); idx != -1 {
			line = line[:idx]
		}
		cleanLines = append(cleanLines, line)
	}
	jsonStr = strings.Join(cleanLines, "\n")

	var route RoutePlan
	if err := json.Unmarshal([]byte(jsonStr), &route); err != nil {
		return RoutePlan{}, err
	}
	return route, nil
}

func extractRoutingGuideSummary(readme string) string {
	start := strings.Index(readme, "## Agents")
	if start == -1 {
		return readme
	}
	return readme[start:]
}
