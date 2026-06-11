package core

import (
	"testing"
	"time"
)

func TestSelectRoutePlan_MarketResearcher(t *testing.T) {
	query := "Phân tích ngành ngân hàng Việt Nam"
	route := SelectRoutePlan(query)

	if route.Agent != "market-researcher" {
		t.Errorf("expected market-researcher, got %s", route.Agent)
	}
}

func TestSelectRoutePlan_EarningsReviewer(t *testing.T) {
	query := "Báo cáo thu nhập quý 2 của VNM"
	route := SelectRoutePlan(query)

	if route.Agent != "earnings-reviewer" {
		t.Errorf("expected earnings-reviewer, got %s", route.Agent)
	}
}

func TestSelectRoutePlan_ModelBuilder(t *testing.T) {
	query := "Định giá DCF cho cổ phiếu FPT"
	route := SelectRoutePlan(query)

	if route.Agent != "model-builder" {
		t.Errorf("expected model-builder, got %s", route.Agent)
	}
}

func TestSelectRoutePlan_MeetingPrep(t *testing.T) {
	query := "Chuẩn bị briefing cho ban lãnh đạo"
	route := SelectRoutePlan(query)

	if route.Agent != "meeting-prep-agent" {
		t.Errorf("expected meeting-prep-agent, got %s", route.Agent)
	}
}

func TestSelectRoutePlan_StatementAuditor(t *testing.T) {
	query := "Nằm ở trang nào trong báo cáo kiểm toán"
	route := SelectRoutePlan(query)

	if route.Agent != "statement-auditor" {
		t.Errorf("expected statement-auditor, got %s", route.Agent)
	}
}

func TestSelectRoutePlan_DefaultFallback(t *testing.T) {
	query := "Xin chào, bạn có khỏe không?"
	route := SelectRoutePlan(query)

	// Default should be market-researcher
	if route.Agent != "market-researcher" {
		t.Errorf("expected market-researcher (default), got %s", route.Agent)
	}
}

func TestSelectRoutePlan_Temporal_Now(t *testing.T) {
	query := "Giá cổ phiếu VCB hôm nay"
	route := SelectRoutePlan(query)

	if route.Temporal.Intent != "realtime" {
		t.Errorf("expected realtime intent, got %s", route.Temporal.Intent)
	}
	expected := time.Now().Format("2006-01-02")
	if route.Temporal.ResolvedDate != expected {
		t.Errorf("expected date %s, got %s", expected, route.Temporal.ResolvedDate)
	}
}

func TestSelectRoutePlan_Temporal_Future(t *testing.T) {
	query := "Dự báo giá cổ phiếu ngày mai"
	route := SelectRoutePlan(query)

	if !route.Temporal.IsFuture {
		t.Error("expected IsFuture=true")
	}
}

func TestSelectRoutePlan_Temporal_Historical2024(t *testing.T) {
	query := "Báo cáo tài chính năm 2024"
	route := SelectRoutePlan(query)

	if route.Temporal.Intent != "historical" {
		t.Errorf("expected historical intent, got %s", route.Temporal.Intent)
	}
	if route.Temporal.ResolvedDate != "2024-12-31" {
		t.Errorf("expected 2024-12-31, got %s", route.Temporal.ResolvedDate)
	}
}

func TestSelectRoutePlan_Temporal_6Months2025(t *testing.T) {
	query := "6 tháng đầu năm 2025"
	route := SelectRoutePlan(query)

	if route.Temporal.Intent != "historical" {
		t.Errorf("expected historical intent, got %s", route.Temporal.Intent)
	}
	if route.Temporal.ResolvedDate != "2025-06-30" {
		t.Errorf("expected 2025-06-30, got %s", route.Temporal.ResolvedDate)
	}
}

func TestSelectRoutePlan_SkillsAssigned(t *testing.T) {
	query := "Phân tích ngành"
	route := SelectRoutePlan(query)

	if len(route.Skills) == 0 {
		t.Error("expected skills to be assigned")
	}
}

func TestSelectRoutePlan_AllowedAgentOnly(t *testing.T) {
	// Ensure only allowed agents are returned
	queries := []string{
		"kiểm toán báo cáo",
		"định giá doanh nghiệp",
		"pitch deck",
		"earnings report",
		"reconciliation",
	}
	for _, q := range queries {
		route := SelectRoutePlan(q)
		if !allowedAgents[route.Agent] {
			t.Errorf("query '%s': agent '%s' not in allowed list", q, route.Agent)
		}
	}
}

func TestHeuristicRoutePlan(t *testing.T) {
	cases := []struct {
		query    string
		expected string
	}{
		{"giá cổ phiếu VCB", "market-researcher"},
		{"báo cáo thu nhập", "earnings-reviewer"},
		{"dcf valuation", "model-builder"},
		{"ban lãnh đạo công ty", "meeting-prep-agent"},
		{"nằm ở trang nào", "statement-auditor"},
	}
	for _, tc := range cases {
		route := heuristicRoutePlan(tc.query, time.Now())
		if route.Agent != tc.expected {
			t.Errorf("query '%s': expected %s, got %s", tc.query, tc.expected, route.Agent)
		}
	}
}

func TestSanitizeRoutePlan_InvalidAgent(t *testing.T) {
	route := RoutePlan{Agent: "nonexistent-agent"}
	sanitized := sanitizeRoutePlan(route)
	if sanitized.Agent != "market-researcher" {
		t.Errorf("expected fallback to market-researcher, got %s", sanitized.Agent)
	}
}

func TestSanitizeRoutePlan_ValidAgent(t *testing.T) {
	route := RoutePlan{Agent: "earnings-reviewer", Skills: []string{"earnings-analysis"}}
	sanitized := sanitizeRoutePlan(route)
	if sanitized.Agent != "earnings-reviewer" {
		t.Errorf("expected earnings-reviewer, got %s", sanitized.Agent)
	}
}

func TestParseRoutePlan(t *testing.T) {
	raw := `{"agent": "earnings-reviewer", "skills": ["earnings-analysis"], "reason": "test"}`
	route, err := parseRoutePlan(raw)
	if err != nil {
		t.Fatalf("parseRoutePlan failed: %v", err)
	}
	if route.Agent != "earnings-reviewer" {
		t.Errorf("expected earnings-reviewer, got %s", route.Agent)
	}
}

func TestParseRoutePlan_WithMarkdown(t *testing.T) {
	raw := "Here is the route: {\"agent\": \"model-builder\"}"
	route, err := parseRoutePlan(raw)
	if err != nil {
		t.Fatalf("parseRoutePlan failed: %v", err)
	}
	if route.Agent != "model-builder" {
		t.Errorf("expected model-builder, got %s", route.Agent)
	}
}

func TestParseRoutePlan_NoJSON(t *testing.T) {
	raw := "no json here"
	_, err := parseRoutePlan(raw)
	if err == nil {
		t.Error("expected error for non-JSON input")
	}
}

func TestGuessSkillsForAgent(t *testing.T) {
	cases := map[string][]string{
		"pitch-agent":        {"pitch-deck"},
		"market-researcher":  {"sector-overview"},
		"earnings-reviewer":  {"earnings-analysis"},
		"model-builder":      {"dcf-model"},
		"month-end-closer":   {"variance-commentary"},
		"unknown-agent":      {},
	}
	for agent, expected := range cases {
		skills := guessSkillsForAgent(agent)
		if len(skills) != len(expected) {
			t.Errorf("agent %s: expected %d skills, got %d", agent, len(expected), len(skills))
			continue
		}
		for i, s := range expected {
			if skills[i] != s {
				t.Errorf("agent %s: expected skill %s, got %s", agent, s, skills[i])
			}
		}
	}
}
