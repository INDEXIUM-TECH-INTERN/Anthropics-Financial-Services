package core

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gemini-cli/internal/domain/entities"
	"gemini-cli/internal/pubsub"
	"gemini-cli/internal/tools"
	"gemini-cli/internal/utils"
)


// BootstrapContext selects the best route plan via the agent's router and
// loads the matching agent + skill documents into the conversation.
func BootstrapContext(agent *entities.Agent) {
	fmt.Printf("🧭 [Context] Initializing agent context...\n")
	ExecuteBootstrap(agent)
}

// ExecuteBootstrap loads agent/skill configuration and appends it to
// the agent's conversation history.
func ExecuteBootstrap(agent *entities.Agent) {
	contextParts := BuildBootstrapContext(agent)
	bootstrapPayload := strings.Join(contextParts, "\n\n")
	agent.AppendUserTextInternal(bootstrapPayload, nil)
}

// BuildBootstrapContext assembles the bootstrap context strings for a given
// agent: agent definition, skill markdown, real-time market data, and suffix.
func BuildBootstrapContext(agent *entities.Agent) []string {
	agentDoc := tools.LoadDocumentWithMetadata("agent", "general")
	logLoadedDocument(agentDoc)

	contextParts := []string{
		fmt.Sprintf("ANTHROPIC AGENT CONFIGURATION\nAgent: General Agent\nMode: Managed Agent (API)"),
		fmt.Sprintf("SYSTEM PROMPT\n%s", agentDoc.Content),
	}

	// TODO: Load skills based on agent capabilities

	if tools.NeedsRealtimeData(agent.UserInput) {
		pubsub.BroadcastLog("Phát hiện nhu cầu dữ liệu Real-time. Đang tìm kiếm...", "process")
		queryPlan := tools.BuildMarketQueryPlan(agent.UserInput)

		// Concurrent search with timeout to prevent indefinite blocking.
		const searchTimeout = 30 * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), searchTimeout)
		defer cancel()

		type searchResult struct {
			result string
			source string
		}
		resultsCh := make(chan searchResult, 2)

		go func() {
			r := tools.SearchGoogle(queryPlan.SearchQuery)
			select {
			case resultsCh <- searchResult{r, "google"}:
			case <-ctx.Done():
			}
		}()
		go func() {
			r := tools.SearchTavily(queryPlan.SearchQuery)
			select {
			case resultsCh <- searchResult{r, "tavily"}:
			case <-ctx.Done():
			}
		}()

		var googleResult, tavilyResult string
		for i := 0; i < 2; i++ {
			select {
			case sr := <-resultsCh:
				if sr.source == "google" {
					googleResult = sr.result
				} else {
					tavilyResult = sr.result
				}
			case <-ctx.Done():
				fmt.Printf("⚠️ [Bootstrap] Search timed out after %v, using partial results\n", searchTimeout)
				i = 2 // break outer loop
			}
		}

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
