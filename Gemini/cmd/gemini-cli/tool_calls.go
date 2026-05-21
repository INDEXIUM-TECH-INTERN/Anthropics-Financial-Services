package main

import (
	"fmt"

	"gemini-cli/internal/models"
	"gemini-cli/internal/tools"
)

func (a *Agent) handleToolCalls(aiMessage models.GeminiContent) bool {
	hasToolCall := false
	for _, part := range aiMessage.Parts {
		a.renderPartText(part)
		if part.FunctionCall == nil {
			continue
		}

		hasToolCall = true
		fmt.Printf("\n🎯 [Action] AI invokes MCP tool: %s\n", part.FunctionCall.Name)
		BroadcastLog(fmt.Sprintf("Thực thi Tool: %s...", part.FunctionCall.Name), "tool")

		result := a.resolveToolCallResult(part.FunctionCall)
		a.appendFunctionResponse(part.FunctionCall, result)
	}
	return hasToolCall
}

func (a *Agent) renderPartText(part models.GeminiPart) {
	if part.Text == "" {
		return
	}

	fmt.Println("\n✨ --- AGENT RESPONSE ---")
	fmt.Println(part.Text)
}

func (a *Agent) resolveToolCallResult(functionCall *models.GeminiFunctionCall) string {
	switch functionCall.Name {
	case "financial_research":
		return a.handleFinancialResearchTool(functionCall.Args)
	case "financial_scrape":
		url, _ := functionCall.Args["url"].(string)
		return tools.ScrapeWeb(url)
	case "financial_calculate":
		expr, _ := functionCall.Args["expression"].(string)
		return tools.Calculate(expr)
	case "handoff_request":
		return a.handleHandoffTool(functionCall.Args)
	case "load_financial_context":
		docType, _ := functionCall.Args["type"].(string)
		docName, _ := functionCall.Args["name"].(string)
		return tools.LoadDocument(docType, docName)
	default:
		return fmt.Sprintf("Error: Unknown tool %s", functionCall.Name)
	}
}

func (a *Agent) handleFinancialResearchTool(args map[string]interface{}) string {
	query, ok := args["query"].(string)
	if !ok {
		return "Error: Missing query parameter"
	}

	searchQuery := query
	if tools.NeedsRealtimeData(a.userInput) {
		searchQuery = tools.BuildMarketQueryPlan(a.userInput).SearchQuery
	}
	return tools.SearchGoogle(searchQuery)
}

func (a *Agent) handleHandoffTool(args map[string]interface{}) string {
	targetAgent, _ := args["target_agent"].(string)
	reason, _ := args["reason"].(string)
	payload, _ := args["task_payload"].(string)

	fmt.Printf("🔀 [Orchestrator] Handoff requested to %s. Reason: %s\n", targetAgent, reason)

	a.handoffPlan = &RoutePlan{
		Agent:  targetAgent,
		Skills: []string{guessInitialSkill(targetAgent)},
		Reason: fmt.Sprintf("Handoff from previous agent: %s. Task: %s", reason, payload),
	}

	return fmt.Sprintf("Successfully initiated handoff to %s.", targetAgent)
}

func guessInitialSkill(agent string) string {
	switch agent {
	case "earnings-reviewer":
		return "earnings-analysis"
	case "market-researcher":
		return "sector-overview"
	default:
		return "general-analysis"
	}
}

func (a *Agent) appendFunctionResponse(functionCall *models.GeminiFunctionCall, result string) {
	a.history = append(a.history, models.GeminiContent{
		Role: "function",
		Parts: []models.GeminiPart{{
			FunctionResponse: &models.GeminiFunctionResponse{
				Name:     functionCall.Name,
				Response: map[string]interface{}{"content": result},
				ID:       functionCall.ID,
			},
		}},
	})
}
