package core

import (
	"context"
	"fmt"
	"strings"

	"gemini-cli/internal/api"
	"gemini-cli/internal/models/messaging"
	"gemini-cli/internal/tools"
	"gemini-cli/internal/utils"
)

type Orchestrator struct {
	agent *Agent
}

func NewOrchestrator(a *Agent) *Orchestrator {
	return &Orchestrator{agent: a}
}

func (o *Orchestrator) ProcessMessage(userInput string) (string, error) {
	o.agent.mu.Lock()
	o.agent.userInput = userInput

	isNewConversation := len(o.agent.conversation.ContextWindow.History) == 0

	if isNewConversation {
		api.BroadcastLog("Khởi tạo cuộc hội thoại mới...", "process")
		if strings.HasPrefix(userInput, "/") {
			if o.handleSlashCommandInternal(userInput) {
				// handled
			}
		} else {
			o.agent.appendUserTextInternal(userInput)
			o.agent.mu.Unlock() // Unlock before calling bootstrapContextInternal to avoid deadlock
			o.bootstrapContextInternal()
			return o.runConversationLoopInternal()
		}
	} else {
		o.agent.appendUserTextInternal(userInput)
	}
	o.agent.mu.Unlock()

	return o.runConversationLoopInternal()
}

func (o *Orchestrator) handleSlashCommandInternal(input string) bool {
	parts := strings.Fields(input)
	cmd := strings.ToLower(parts[0])
	args := strings.Join(parts[1:], " ")

	var route RoutePlan
	switch cmd {
	case "/earnings":
		api.BroadcastLog("Kích hoạt lệnh /earnings...", "routing")
		route = RoutePlan{Agent: "earnings-reviewer", Skills: []string{"earnings-analysis"}, Reason: "Slash command /earnings"}
	case "/market":
		api.BroadcastLog("Kích hoạt lệnh /market...", "routing")
		route = RoutePlan{Agent: "market-researcher", Skills: []string{"sector-overview"}, Reason: "Slash command /market"}
	case "/help":
		fmt.Println("\n-- ANTHROPIC CLI SIMULATOR COMMANDS --")
		fmt.Println("/earnings <ticker> : Run Earnings Reviewer workflow")
		fmt.Println("/market <query>   : Run Market Researcher workflow")
		return false
	default:
		return false
	}

	o.agent.userInput = args
	o.agent.appendUserTextInternal(args)
	o.executeBootstrapWithRouteInternal(route)
	return true
}

func (o *Orchestrator) bootstrapContextInternal() {
	route := o.agent.selectRoutePlan()
	fmt.Printf("🧭 [Router] Identified Agent: %s (Reason: %s)\n", route.Agent, route.Reason)
	o.executeBootstrapWithRouteInternal(route)
}

func (o *Orchestrator) executeBootstrapWithRouteInternal(route RoutePlan) {
	api.BroadcastLog(fmt.Sprintf("Nạp cấu hình cho Agent: %s...", route.Agent), "routing")
	fmt.Printf("🧭 [Context] Orchestrator: Loading %s configuration...\n", route.Agent)
	contextParts := o.buildBootstrapContextInternal(route)
	bootstrapPayload := strings.Join(contextParts, "\n\n")
	o.agent.appendUserTextInternal(bootstrapPayload)
}

func (o *Orchestrator) buildBootstrapContextInternal(route RoutePlan) []string {
	agentDoc := tools.LoadDocumentWithMetadata("agent", route.Agent)
	o.logLoadedDocument(agentDoc)

	contextParts := []string{
		fmt.Sprintf("ANTHROPIC AGENT CONFIGURATION\nAgent: %s\nSkills: %s\nMode: Managed Agent (API)", route.Agent, strings.Join(route.Skills, ", ")),
		fmt.Sprintf("SYSTEM PROMPT (from agents/%s.md)\n%s", route.Agent, agentDoc.Content),
	}

	maxSkills := 1
	for i, skill := range route.Skills {
		if i >= maxSkills {
			fmt.Printf("⚠️ [Context] Bỏ qua skill %s để tối ưu hóa token.\n", skill)
			continue
		}
		api.BroadcastLog(fmt.Sprintf("Đang nạp skill chuyên biệt: %s", skill), "process")
		skillDoc := tools.LoadDocumentWithMetadata("skill", route.Agent+"/"+skill)
		o.logLoadedDocument(skillDoc)

		content := skillDoc.Content
		if len(content) > 4000 {
			content = content[:4000] + "\n... [Nội dung bị cắt bớt để tối ưu hóa context]"
		}
		contextParts = append(contextParts, fmt.Sprintf("SKILL MARKDOWN (%s)\n%s", skill, content))
	}

	if tools.NeedsRealtimeData(o.agent.userInput) {
		api.BroadcastLog("Phát hiện nhu cầu dữ liệu Real-time. Đang tìm kiếm...", "process")
		queryPlan := tools.BuildMarketQueryPlan(o.agent.userInput)
		realtimeResult := tools.SearchGoogle(queryPlan.SearchQuery)
		contextParts = append(contextParts, fmt.Sprintf("REAL-TIME MARKET DATA\n%s", realtimeResult))
	}

	contextParts = append(contextParts, utils.LoadPrompt("bootstrap_context_suffix.txt"))
	return contextParts
}

func (o *Orchestrator) runConversationLoopInternal() (string, error) {
	for {
		o.agent.mu.Lock()
		systemPrompt := o.agent.systemPrompt
		history := make([]messaging.Message, len(o.agent.conversation.ContextWindow.History))
		copy(history, o.agent.conversation.ContextWindow.History)
		tools := o.agent.dispatcher.GetTools()
		o.agent.mu.Unlock()

		var messages []messaging.Message
		if systemPrompt != "" {
			messages = append(messages, messaging.Message{
				Role:    messaging.RoleSystem,
				Content: systemPrompt,
			})
		}
		messages = append(messages, history...)

		req := messaging.Request{
			History: messages,
			Tools:   tools,
		}

		aiMessage, err := o.agent.provider.Generate(context.Background(), req)
		if err != nil {
			return "", err
		}

		o.agent.mu.Lock()
		o.agent.conversation.ContextWindow.History = append(o.agent.conversation.ContextWindow.History, aiMessage)
		hasToolCall := o.agent.dispatcher.HandleToolCalls(aiMessage)

		if o.agent.handoffPlan != nil {
			plan := *o.agent.handoffPlan
			o.agent.handoffPlan = nil
			fmt.Printf("\n🔀 [Orchestrator] Executing handoff to: %s\n", plan.Agent)
			o.executeBootstrapWithRouteInternal(plan)
			o.agent.mu.Unlock()
			continue
		}
		o.agent.mu.Unlock()

		if !hasToolCall {
			return extractResponseText(aiMessage), nil
		}
	}
}

func (o *Orchestrator) logLoadedDocument(doc tools.LoadedDocument) {
	fmt.Printf("📎 [Sync] %s: %s (Size: %d chars)\n", strings.ToUpper(doc.DocType), doc.Name, len(doc.Content))
}

func extractResponseText(aiMessage messaging.Message) string {
	return aiMessage.Content
}
