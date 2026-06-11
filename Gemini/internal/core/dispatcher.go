package core

import (
	"fmt"
	"strings"
	"sync"

	"gemini-cli/internal/api"
	"gemini-cli/internal/models/messaging"
	"gemini-cli/internal/tools"
)

const maxCacheEntries = 200 // prevent unbounded memory growth

type Dispatcher struct {
	agent         *Agent
	researchCache map[string]string // cache query -> result để giảm gọi tool lặp lại (giảm quota)
	cacheOrder    []string          // LRU eviction order (oldest first)
	mu            sync.RWMutex
}

func NewDispatcher(a *Agent) *Dispatcher {
	return &Dispatcher{
		agent:         a,
		researchCache: make(map[string]string),
		cacheOrder:    make([]string, 0, maxCacheEntries),
	}
}

// evictCacheIfNeeded removes oldest entries when cache exceeds maxCacheEntries.
// Must be called with d.mu held (write lock).
func (d *Dispatcher) evictCacheIfNeeded() {
	for len(d.cacheOrder) > maxCacheEntries {
		oldest := d.cacheOrder[0]
		d.cacheOrder = d.cacheOrder[1:]
		delete(d.researchCache, oldest)
	}
}

func (d *Dispatcher) GetTools() []messaging.ToolSchema {
	return []messaging.ToolSchema{
		d.newFinancialResearchTool(),
		d.newTavilySearchTool(),
		d.newFinancialScrapeTool(),
		d.newFinancialCalculateTool(),
		d.newHandoffRequestTool(),
		d.newLoadContextTool(),
	}
}

func (d *Dispatcher) HandleToolCalls(aiMessage messaging.Message) bool {
	if len(aiMessage.ToolCalls) == 0 {
		return false
	}

	for _, toolCall := range aiMessage.ToolCalls {
		fmt.Printf("🎯 [Action] AI invokes MCP tool: %s\n", toolCall.Name)
		api.BroadcastEvent(fmt.Sprintf("Thực thi Tool: %s...", toolCall.Name), "tool_executed", map[string]interface{}{
			"tool": toolCall.Name,
			"args": toolCall.Args,
		})

		result := d.resolveToolCallResult(&toolCall)
		d.appendFunctionResponse(&toolCall, result)
	}
	return true
}

func (d *Dispatcher) resolveToolCallResult(toolCall *messaging.ToolCall) string {
	switch toolCall.Name {
	case "financial_research":
		query, _ := toolCall.Args["query"].(string)
		api.BroadcastLog(fmt.Sprintf("Tra cứu dữ liệu (Google): %s", query), "tool")
		return d.handleFinancialResearchTool(toolCall.Args)
	case "tavily_search":
		query, _ := toolCall.Args["query"].(string)
		api.BroadcastLog(fmt.Sprintf("Tra cứu dữ liệu (Tavily): %s", query), "tool")
		return d.handleTavilySearchTool(toolCall.Args)
	case "financial_scrape":
		url, _ := toolCall.Args["url"].(string)
		api.BroadcastLog(fmt.Sprintf("Đang đọc nội dung từ: %s", url), "tool")
		return d.handleFinancialScrapeTool(toolCall.Args)
	case "financial_calculate":
		expr, _ := toolCall.Args["expression"].(string)
		api.BroadcastLog(fmt.Sprintf("Thực hiện tính toán: %s", expr), "tool")
		return tools.Calculate(expr)
	case "handoff_request":
		target, _ := toolCall.Args["target_agent"].(string)
		api.BroadcastLog(fmt.Sprintf("Yêu cầu chuyển giao sang: %s", target), "routing")
		return d.handleHandoffTool(toolCall.Args)
	case "load_financial_context":
		docType, _ := toolCall.Args["type"].(string)
		docName, _ := toolCall.Args["name"].(string)
		api.BroadcastLog(fmt.Sprintf("Nạp tài liệu: %s/%s", docType, docName), "process")
		return tools.LoadDocument(docType, docName)
	default:
		return fmt.Sprintf("Error: Unknown tool %s", toolCall.Name)
	}
}

func (d *Dispatcher) handleFinancialResearchTool(args map[string]interface{}) string {
	query, ok := args["query"].(string)
	if !ok {
		return "Error: Missing query parameter"
	}

	searchQuery := query
	if tools.NeedsRealtimeData(d.agent.userInput) {
		searchQuery = tools.BuildMarketQueryPlan(d.agent.userInput).SearchQuery
	}

	// Simple cache để tránh gọi lại cùng query nhiều lần (rất hay xảy ra trong ReAct loop)
	key := strings.ToLower(strings.TrimSpace(searchQuery))
	d.mu.RLock()
	cached, ok := d.researchCache["google_"+key]
	d.mu.RUnlock()
	if ok && cached != "" {
		fmt.Printf("💾 [Cache] Dùng kết quả research đã cache cho: %s\n", searchQuery)
		api.BroadcastLog("Dùng kết quả tìm kiếm từ cache (tiết kiệm quota)", "success")
		return cached
	}

	result := tools.SearchGoogle(searchQuery)
	if result != "" && !strings.HasPrefix(result, "Lỗi") && !strings.HasPrefix(result, "Error") {
		d.mu.Lock()
		d.researchCache["google_"+key] = result
		d.cacheOrder = append(d.cacheOrder, "google_"+key)
		d.evictCacheIfNeeded()
		d.mu.Unlock()
	}
	return result
}

func (d *Dispatcher) handleTavilySearchTool(args map[string]interface{}) string {
	query, ok := args["query"].(string)
	if !ok {
		return "Error: Missing query parameter"
	}

	searchQuery := query
	if tools.NeedsRealtimeData(d.agent.userInput) {
		searchQuery = tools.BuildMarketQueryPlan(d.agent.userInput).SearchQuery
	}

	key := strings.ToLower(strings.TrimSpace(searchQuery))
	d.mu.RLock()
	cached, ok := d.researchCache["tavily_"+key]
	d.mu.RUnlock()
	if ok && cached != "" {
		fmt.Printf("💾 [Cache] Dùng kết quả research đã cache (Tavily) cho: %s\n", searchQuery)
		api.BroadcastLog("Dùng kết quả tìm kiếm Tavily từ cache (tiết kiệm quota)", "success")
		return cached
	}

	result := tools.SearchTavily(searchQuery)
	if result != "" && !strings.HasPrefix(result, "Lỗi") && !strings.HasPrefix(result, "Error") {
		d.mu.Lock()
		d.researchCache["tavily_"+key] = result
		d.cacheOrder = append(d.cacheOrder, "tavily_"+key)
		d.evictCacheIfNeeded()
		d.mu.Unlock()
	}
	return result
}

func (d *Dispatcher) handleHandoffTool(args map[string]interface{}) string {
	targetAgent, _ := args["target_agent"].(string)
	reason, _ := args["reason"].(string)
	payload, _ := args["task_payload"].(string)

	fmt.Printf("🔀 [Orchestrator] Handoff requested to %s. Reason: %s\n", targetAgent, reason)

	d.agent.handoffPlan = &RoutePlan{
		Agent:  targetAgent,
		Skills: guessSkillsForAgent(targetAgent),
		Reason: fmt.Sprintf("Handoff from previous agent: %s. Task: %s", reason, payload),
	}

	return fmt.Sprintf("Successfully initiated handoff to %s.", targetAgent)
}

func (d *Dispatcher) appendFunctionResponse(toolCall *messaging.ToolCall, result string) {
	d.agent.conversation.ContextWindow.History = append(d.agent.conversation.ContextWindow.History, messaging.Message{
		Role: messaging.RoleTool,
		ToolResponses: []messaging.ToolResponse{{
			CallID:  toolCall.ID,
			Name:    toolCall.Name,
			Content: result,
		}},
	})
}

func (d *Dispatcher) newFinancialResearchTool() messaging.ToolSchema {
	return messaging.ToolSchema{
		Name:        "financial_research",
		Description: "Truy vấn dữ liệu thị trường thực tế bằng Google Search (tương đương MCP Data Connectors).",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{"type": "string", "description": "Từ khóa hoặc mã chứng khoán cần tra cứu"},
			},
			"required": []string{"query"},
		},
	}
}

func (d *Dispatcher) newTavilySearchTool() messaging.ToolSchema {
	return messaging.ToolSchema{
		Name:        "tavily_search",
		Description: "Truy vấn dữ liệu thị trường thực tế bằng Tavily Search (phù hợp để tìm thông tin tài chính chuyên sâu, câu trả lời trực tiếp).",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{"type": "string", "description": "Từ khóa hoặc mã chứng khoán cần tra cứu"},
			},
			"required": []string{"query"},
		},
	}
}

func (d *Dispatcher) newFinancialScrapeTool() messaging.ToolSchema {
	return messaging.ToolSchema{
		Name:        "financial_scrape",
		Description: "Đọc sâu nội dung báo cáo, tin tức từ URL (tương đương Web/PDF Reader MCP).",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url": map[string]interface{}{"type": "string", "description": "URL nguồn tài liệu"},
			},
			"required": []string{"url"},
		},
	}
}

func (d *Dispatcher) newFinancialCalculateTool() messaging.ToolSchema {
	return messaging.ToolSchema{
		Name:        "financial_calculate",
		Description: "Thực hiện tính toán tài chính (tương đương Python/Excel tools).",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"expression": map[string]interface{}{"type": "string", "description": "Biểu thức toán học (ví dụ: DCF, P/E calculation)"},
			},
			"required": []string{"expression"},
		},
	}
}

func (d *Dispatcher) newHandoffRequestTool() messaging.ToolSchema {
	return messaging.ToolSchema{
		Name:        "handoff_request",
		Description: "Gửi yêu cầu chuyển giao tác vụ cho Agent chuyên môn khác (tương đương orchestrate.py).",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"target_agent": map[string]interface{}{"type": "string", "description": "Agent mục tiêu (ví dụ: earnings-reviewer, model-builder)"},
				"reason":       map[string]interface{}{"type": "string", "description": "Lý do và ngữ cảnh cần chuyển giao"},
				"task_payload": map[string]interface{}{"type": "string", "description": "Dữ liệu/Nhiệm vụ cụ thể cần thực hiện"},
			},
			"required": []string{"target_agent", "reason"},
		},
	}
}

func (d *Dispatcher) newLoadContextTool() messaging.ToolSchema {
	return messaging.ToolSchema{
		Name:        "load_financial_context",
		Description: "Nạp tài liệu kỹ năng (Skill) từ bộ Vertical Plugins.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"type": map[string]interface{}{"type": "string", "description": "agent hoặc skill"},
				"name": map[string]interface{}{"type": "string", "description": "Tên tài liệu (e.g. market-researcher/sector-overview)"},
			},
			"required": []string{"type", "name"},
		},
	}
}

func (d *Dispatcher) handleFinancialScrapeTool(args map[string]interface{}) string {
	url, ok := args["url"].(string)
	if !ok {
		return "Error: Missing url parameter"
	}

	key := strings.ToLower(strings.TrimSpace(url))
	d.mu.RLock()
	cached, ok := d.researchCache["scrape_"+key]
	d.mu.RUnlock()
	if ok && cached != "" {
		fmt.Printf("💾 [Cache] Dùng kết quả scrape đã cache cho: %s\n", url)
		api.BroadcastLog("Dùng kết quả đọc nội dung từ cache (tiết kiệm quota)", "success")
		return cached
	}

	result := tools.ScrapeWeb(url)
	if result != "" && !strings.HasPrefix(result, "Lỗi") && !strings.HasPrefix(result, "Error") {
		d.mu.Lock()
		d.researchCache["scrape_"+key] = result
		d.cacheOrder = append(d.cacheOrder, "scrape_"+key)
		d.evictCacheIfNeeded()
		d.mu.Unlock()
	}
	return result
}
