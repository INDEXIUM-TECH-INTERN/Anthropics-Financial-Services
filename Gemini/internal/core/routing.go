package core

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"gemini-cli/internal/pubsub"
	"gemini-cli/internal/routing"
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
	pubsub.BroadcastLog("Đang phân tích yêu cầu để chọn Agent tối ưu...", "process")
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

	pubsub.BroadcastEvent(
		fmt.Sprintf("Đã chọn Agent: %s (Lý do: %s)", route.Agent, route.Reason),
		"agent_selected",
		map[string]interface{}{
			"agent":  route.Agent,
			"reason": route.Reason,
			"skills": route.Skills,
		},
	)
	return sanitizeRoutePlan(route)
}

func heuristicRoutePlan(userInput string, now time.Time) RoutePlan {
	q := strings.ToLower(userInput)
	var route RoutePlan
	route.Agent = "market-researcher"

	// Determine Agent
	if utils.ContainsAny(q, "ban lãnh đạo", "lãnh đạo", "ban lanh dao", "board of directors", "leadership", "executive", "ban điều hành", "ban dieu hanh", "hội đồng quản trị", "hoi dong quan tri") {
		route.Agent = "meeting-prep-agent"
	} else if utils.ContainsAny(q, "nằm ở trang nào", "nam o trang nao", "trang mấy", "trang may", "audit", "kiểm toán", "kiem toan") {
		route.Agent = "statement-auditor"
	} else if utils.ContainsAny(q, "báo cáo thu nhập", "bao cao thu nhap", "trích dẫn báo cáo", "trich dan bao cao", "6 tháng đầu năm", "6 thang dau nam", "quý", "quy", "doanh thu", "lợi nhuận") {
		route.Agent = "earnings-reviewer"
	} else if utils.ContainsAny(q, "10 năm qua", "10 nam qua", "dự phóng", "du phong", "dcf-model", "định giá", "valuation", "lbo-model") {
		route.Agent = "model-builder"
	} else if utils.ContainsAny(q, "so sánh", "so sanh", "phân tích kỹ thuật", "phan tich ky thuat", "giá cổ phiếu", "gia co phieu", "lợi thế gì", "loi the gi", "ngành ngân hàng", "nganh ngan hang") {
		route.Agent = "market-researcher"
	} else if utils.ContainsAny(q, "báo cáo tài chính năm 2024", "báo cáo tài chính năm 2023", "báo cáo năm") {
		route.Agent = "earnings-reviewer"
	}

	// 2. Determine Temporal Intent & Date
	temporal := routing.ResolveTemporal(q, now)
	route.Temporal.Intent = temporal.Intent
	route.Temporal.ResolvedDate = temporal.ResolvedDate
	route.Temporal.IsFuture = temporal.IsFuture

	route.Skills = guessSkillsForAgent(route.Agent)
	route.Reason = "Heuristically determined based on query patterns."
	return route
}

func guessSkillsForAgent(agent string) []string {
	switch agent {
	case "pitch-agent":
		return []string{"pitch-deck"}
	case "meeting-prep-agent":
		return []string{}
	case "market-researcher":
		return []string{"sector-overview"}
	case "earnings-reviewer":
		return []string{"earnings-analysis"}
	case "model-builder":
		return []string{"dcf-model"}
	case "valuation-reviewer":
		return []string{}
	case "gl-reconciler":
		return []string{}
	case "month-end-closer":
		return []string{"variance-commentary"}
	case "statement-auditor":
		return []string{}
	case "kyc-screener":
		return []string{}
	default:
		return []string{}
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

func sanitizeRoutePlan(route RoutePlan) RoutePlan {
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
