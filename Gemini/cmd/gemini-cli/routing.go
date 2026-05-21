package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gemini-cli/internal/tools"
	"gemini-cli/internal/utils"
)

type RoutePlan struct {
	Agent    string `json:"agent"`
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

func (a *Agent) selectRoutePlan() RoutePlan {
	routerSystemPrompt := utils.LoadPrompt("router_system_prompt.txt")
	routerUserPrompt := buildRouterUserPrompt(a.userInput, tools.LoadRoutingGuide(), time.Now())

	raw, err := a.routeWithProviderFallback(routerSystemPrompt, routerUserPrompt)
	if err != nil {
		return fallbackRoutePlan(a.userInput)
	}

	fmt.Printf("\n[DEBUG] Raw AI Response: %s\n", raw)
	route, err := parseRoutePlan(raw)
	if err != nil {
		fmt.Printf("⚠️ [Router] Parse error: %v. Using fallback.\n", err)
		return fallbackRoutePlan(a.userInput)
	}

	fmt.Printf("\n🤖 [Router Response]\nAgent: %s\nSkills: %v\nTemporal: %+v\nReason: %s\n\n", route.Agent, route.Skills, route.Temporal, route.Reason)
	return sanitizeRoutePlan(route, a.userInput)
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
	// Lượt 1: Gemini
	raw, err := a.geminiProvider.GenerateText(systemPrompt, userPrompt)
	if err == nil {
		return raw, nil
	}

	// Lượt 2: SambaNova
	raw, err = a.sambanovaProvider.GenerateText(systemPrompt, userPrompt)
	if err == nil {
		return raw, nil
	}

	// Lượt 3: Groq
	raw, err = a.groqProvider.GenerateText(systemPrompt, userPrompt)
	if err == nil {
		return raw, nil
	}

	// Lượt 4: OpenRouter
	return a.openrouterProvider.GenerateText(systemPrompt, userPrompt)
}


func fallbackRoutePlan(userInput string) RoutePlan {
	return RoutePlan{
		Agent:  "market-researcher",
		Skills: []string{"sector-overview"},
		Reason: "Default fallback for general requests.",
	}
}

func sanitizeRoutePlan(route RoutePlan, userInput string) RoutePlan {
	if !allowedAgents[route.Agent] {
		return fallbackRoutePlan(userInput)
	}

	validSkills := filterValidSkills(route.Agent, route.Skills)
	if len(validSkills) == 0 {
		route.Skills = []string{guessInitialSkill(route.Agent)}
	} else {
		route.Skills = validSkills
	}

	return route
}

func filterValidSkills(agentName string, skills []string) []string {
	allowedSkills := allowedSkillsByAgent[agentName]
	var validSkills []string

	for _, skill := range skills {
		if allowedSkills[skill] {
			validSkills = append(validSkills, skill)
		}
	}
	return validSkills
}

func parseRoutePlan(raw string) (RoutePlan, error) {
	// Tìm JSON trong chuỗi văn bản (đề phòng AI trả về text thừa)
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start == -1 || end == -1 || end < start {
		return RoutePlan{}, fmt.Errorf("no JSON found")
	}

	var route RoutePlan
	if err := json.Unmarshal([]byte(raw[start:end+1]), &route); err != nil {
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
