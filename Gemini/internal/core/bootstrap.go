package core

import (
	"fmt"
	"strings"
	"sync"

	"gemini-cli/internal/pubsub"
	"gemini-cli/internal/tools"
	"gemini-cli/internal/utils"
)

// BootstrapContext selects the best route plan via the agent's router and
// loads the matching agent + skill documents into the conversation.
func BootstrapContext(agent *Agent) {
	route := agent.selectRoutePlan()
	fmt.Printf("🧭 [Router] Identified Agent: %s (Reason: %s)\n", route.Agent, route.Reason)
	ExecuteBootstrapWithRoute(agent, route)
}

// ExecuteBootstrapWithRoute loads agent/skill configuration and appends it to
// the agent's conversation history.
func ExecuteBootstrapWithRoute(agent *Agent, route RoutePlan) {
	pubsub.BroadcastLog(fmt.Sprintf("Nạp cấu hình cho Agent: %s...", route.Agent), "routing")
	fmt.Printf("🧭 [Context] Orchestrator: Loading %s configuration...\n", route.Agent)
	contextParts := BuildBootstrapContext(agent, route)
	bootstrapPayload := strings.Join(contextParts, "\n\n")
	agent.appendUserTextInternal(bootstrapPayload, nil)
}

// BuildBootstrapContext assembles the bootstrap context strings for a given
// route: agent definition, skill markdown, real-time market data, and suffix.
func BuildBootstrapContext(agent *Agent, route RoutePlan) []string {
	agentDoc := tools.LoadDocumentWithMetadata("agent", route.Agent)
	logLoadedDocument(agentDoc)

	contextParts := []string{
		fmt.Sprintf("ANTHROPIC AGENT CONFIGURATION\nAgent: %s\nSkills: %s\nMode: Managed Agent (API)", route.Agent, strings.Join(route.Skills, ", ")),
		fmt.Sprintf("SYSTEM PROMPT (from agents/%s.md)\n%s", route.Agent, agentDoc.Content),
	}

	// Nạp tất cả các skills hợp lệ từ route plan
	for _, skill := range route.Skills {
		pubsub.BroadcastEvent(fmt.Sprintf("Đang nạp skill chuyên biệt: %s", skill), "skill_loaded", map[string]interface{}{
			"skill": skill,
		})
		skillDoc := tools.LoadDocumentWithMetadata("skill", route.Agent+"/"+skill)

		if strings.HasPrefix(skillDoc.Content, "Lỗi:") {
			fmt.Printf("⚠️ [Context] Không thể nạp skill %s: %s\n", skill, skillDoc.Content)
			continue
		}

		logLoadedDocument(skillDoc)

		content := skillDoc.Content
		// Giới hạn độ dài để tránh tràn context nếu skill quá lớn
		if len(content) > 8000 {
			content = content[:8000] + "\n... [Nội dung bị cắt bớt để tối ưu hóa context]"
		}
		contextParts = append(contextParts, fmt.Sprintf("SKILL MARKDOWN (%s)\n%s", skill, content))
	}

	if tools.NeedsRealtimeData(agent.userInput) {
		pubsub.BroadcastLog("Phát hiện nhu cầu dữ liệu Real-time. Đang tìm kiếm...", "process")
		queryPlan := tools.BuildMarketQueryPlan(agent.userInput)

		var googleResult, tavilyResult string
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			googleResult = tools.SearchGoogle(queryPlan.SearchQuery)
		}()
		go func() {
			defer wg.Done()
			tavilyResult = tools.SearchTavily(queryPlan.SearchQuery)
		}()
		wg.Wait()

		var combinedResults []string
		if googleResult != "" && !strings.HasPrefix(googleResult, "Lỗi") && !strings.HasPrefix(googleResult, "Error") {
			combinedResults = append(combinedResults, "--- TỪ GOOGLE SEARCH ---\n"+googleResult)
		}
		if tavilyResult != "" && !strings.HasPrefix(tavilyResult, "Lỗi") && !strings.HasPrefix(tavilyResult, "Error") {
			combinedResults = append(combinedResults, "--- TỪ TAVILY SEARCH ---\n"+tavilyResult)
		}

		if len(combinedResults) > 0 {
			contextParts = append(contextParts, fmt.Sprintf("REAL-TIME MARKET DATA\n%s", strings.Join(combinedResults, "\n\n")))
		}
	}

	contextParts = append(contextParts, utils.LoadPrompt("bootstrap_context_suffix.txt"))
	return contextParts
}

// logLoadedDocument prints a log line for a loaded document.
func logLoadedDocument(doc tools.LoadedDocument) {
	fmt.Printf("📎 [Sync] %s: %s (Size: %d chars)\n", strings.ToUpper(doc.DocType), doc.Name, len(doc.Content))
}
