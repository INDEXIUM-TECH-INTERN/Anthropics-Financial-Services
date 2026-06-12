package core

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gemini-cli/internal/cache"
	"gemini-cli/internal/models/messaging"
	"gemini-cli/internal/pubsub"
	"gemini-cli/internal/scripts/report_gen"
	"gemini-cli/internal/tools"
	"gemini-cli/internal/tools/handlers"
	"gemini-cli/internal/utils"
)

const maxCacheEntries = 200 // prevent unbounded memory growth

// Dispatcher handles tool execution with LRU caching and map-based dispatch.
// Tools are registered in a map for O(1) lookup, replacing the previous
// switch-case approach (Open/Closed Principle compliance).
type Dispatcher struct {
	agent    *Agent
	cache    *cache.LRUCache
	handlers map[string]toolHandler
}

// toolHandler is a function type that executes a tool and returns a string result.
type toolHandler func(args handlers.Args) string

// NewDispatcher creates a dispatcher with all tools registered.
func NewDispatcher(a *Agent) *Dispatcher {
	d := &Dispatcher{
		agent:    a,
		cache:    cache.NewLRUCache(maxCacheEntries),
		handlers: make(map[string]toolHandler, 8),
	}
	d.registerHandlers()
	return d
}

// registerHandlers maps tool names to their handler functions.
// To add a new tool: implement the handler method and add one line here.
func (d *Dispatcher) registerHandlers() {
	d.handlers["financial_research"] = d.handleFinancialResearch
	d.handlers["tavily_search"] = d.handleTavilySearch
	d.handlers["financial_scrape"] = d.handleFinancialScrape
	d.handlers["financial_calculate"] = d.handleFinancialCalculate
	d.handlers["handoff_request"] = d.handleHandoff
	d.handlers["load_financial_context"] = d.handleLoadContext
	d.handlers["export_report"] = d.handleExportReport
	d.handlers["read_local_file"] = d.handleReadLocalFile
}

// GetTools returns the list of available tool schemas for LLM function calling.
func (d *Dispatcher) GetTools() []messaging.ToolSchema {
	return []messaging.ToolSchema{
		{
			Name:        "financial_research",
			Description: "Truy vấn dữ liệu thị trường thực tế bằng Google Search (tương đương MCP Data Connectors).",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{"type": "string", "description": "Từ khóa hoặc mã chứng khoán cần tra cứu"},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "tavily_search",
			Description: "Truy vấn dữ liệu thị trường thực tế bằng Tavily Search (phù hợp để tìm thông tin tài chính chuyên sâu, câu trả lời trực tiếp).",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{"type": "string", "description": "Từ khóa hoặc mã chứng khoán cần tra cứu"},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "financial_scrape",
			Description: "Đọc sâu nội dung báo cáo, tin tức từ URL (tương đương Web/PDF Reader MCP).",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url": map[string]interface{}{"type": "string", "description": "URL nguồn tài liệu"},
				},
				"required": []string{"url"},
			},
		},
		{
			Name:        "financial_calculate",
			Description: "Thực hiện tính toán tài chính (tương đương Python/Excel tools).",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"expression": map[string]interface{}{"type": "string", "description": "Biểu thức toán học (ví dụ: DCF, P/E calculation)"},
				},
				"required": []string{"expression"},
			},
		},
		{
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
		},
		{
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
		},
		{
			Name:        "export_report",
			Description: "Xuất báo cáo tài chính chuyên nghiệp dưới dạng Excel (.xlsx) hoặc PowerPoint (.pptx).",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"format": map[string]interface{}{
						"type":        "string",
						"description": "Định dạng báo cáo: 'xlsx' hoặc 'pptx'",
						"enum":        []string{"xlsx", "pptx"},
					},
					"title": map[string]interface{}{
						"type":        "string",
						"description": "Tiêu đề của báo cáo",
					},
					"data": map[string]interface{}{
						"type":        "string",
						"description": "Dữ liệu JSON thô để đưa vào báo cáo (optional).",
					},
				},
				"required": []string{"format", "title"},
			},
		},
		{
			Name:        "read_local_file",
			Description: "Đọc nội dung tệp tin cục bộ trên server (ví dụ: các tệp báo cáo đã xuất trong thư mục exports).",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{"type": "string", "description": "Tên tệp hoặc đường dẫn tệp (ví dụ: report_123.xlsx)"},
				},
				"required": []string{"path"},
			},
		},
	}
}

// HandleToolCalls processes tool calls from an AI response message using map-based dispatch.
// Returns true if any tool was executed.
func (d *Dispatcher) HandleToolCalls(aiMessage messaging.Message) bool {
	if len(aiMessage.ToolCalls) == 0 {
		return false
	}

	for _, toolCall := range aiMessage.ToolCalls {
		fmt.Printf("🎯 [Action] AI invokes MCP tool: %s\n", toolCall.Name)
		pubsub.BroadcastEvent(fmt.Sprintf("Thực thi Tool: %s...", toolCall.Name), "tool_executed", map[string]interface{}{
			"tool": toolCall.Name,
			"args": toolCall.Args,
		})

		result := d.dispatchToolCall(&toolCall)
		d.appendFunctionResponse(&toolCall, result)
	}
	return true
}

// dispatchToolCall looks up the handler in the map and executes it.
func (d *Dispatcher) dispatchToolCall(toolCall *messaging.ToolCall) string {
	handler, ok := d.handlers[toolCall.Name]
	if !ok {
		return fmt.Sprintf("Error: Unknown tool %s", toolCall.Name)
	}
	return handler(toolCall.Args)
}

// ─── Tool Handler Implementations ───────────────────────────────────────────

func (d *Dispatcher) handleFinancialResearch(args handlers.Args) string {
	query, _ := args["query"].(string)
	if query == "" {
		return "Error: Missing query parameter"
	}
	pubsub.BroadcastLog(fmt.Sprintf("Tra cứu dữ liệu (Google): %s", query), "tool")

	searchQuery := query
	if tools.NeedsRealtimeData(d.agent.userInput) {
		searchQuery = tools.BuildMarketQueryPlan(d.agent.userInput).SearchQuery
	}

	if cached, ok := d.cacheGet("google", searchQuery); ok && cached != "" {
		fmt.Printf("💾 [Cache] Dùng kết quả research đã cache cho: %s\n", searchQuery)
		pubsub.BroadcastLog("Dùng kết quả tìm kiếm từ cache (tiết kiệm quota)", "success")
		return cached
	}

	result := tools.SearchGoogle(searchQuery)
	if result != "" && !strings.HasPrefix(result, "Lỗi") && !strings.HasPrefix(result, "Error") {
		d.cachePut("google", searchQuery, result)
	}
	return result
}

func (d *Dispatcher) handleTavilySearch(args handlers.Args) string {
	query, _ := args["query"].(string)
	if query == "" {
		return "Error: Missing query parameter"
	}
	pubsub.BroadcastLog(fmt.Sprintf("Tra cứu dữ liệu (Tavily): %s", query), "tool")

	searchQuery := query
	if tools.NeedsRealtimeData(d.agent.userInput) {
		searchQuery = tools.BuildMarketQueryPlan(d.agent.userInput).SearchQuery
	}

	if cached, ok := d.cacheGet("tavily", searchQuery); ok && cached != "" {
		fmt.Printf("💾 [Cache] Dùng kết quả research đã cache (Tavily) cho: %s\n", searchQuery)
		pubsub.BroadcastLog("Dùng kết quả tìm kiếm Tavily từ cache (tiết kiệm quota)", "success")
		return cached
	}

	result := tools.SearchTavily(searchQuery)
	if result != "" && !strings.HasPrefix(result, "Lỗi") && !strings.HasPrefix(result, "Error") {
		d.cachePut("tavily", searchQuery, result)
	}
	return result
}

func (d *Dispatcher) handleFinancialScrape(args handlers.Args) string {
	url, _ := args["url"].(string)
	if url == "" {
		return "Error: Missing url parameter"
	}
	pubsub.BroadcastLog(fmt.Sprintf("Đang đọc nội dung từ: %s", url), "tool")

	if cached, ok := d.cacheGet("scrape", url); ok && cached != "" {
		fmt.Printf("💾 [Cache] Dùng kết quả scrape đã cache cho: %s\n", url)
		pubsub.BroadcastLog("Dùng kết quả đọc nội dung từ cache (tiết kiệm quota)", "success")
		return cached
	}

	result := tools.ScrapeWeb(url)
	if result != "" && !strings.HasPrefix(result, "Lỗi") && !strings.HasPrefix(result, "Error") {
		d.cachePut("scrape", url, result)
	}
	return result
}

func (d *Dispatcher) handleFinancialCalculate(args handlers.Args) string {
	expr, _ := args["expression"].(string)
	if expr == "" {
		return "Error: Missing expression parameter"
	}
	pubsub.BroadcastLog(fmt.Sprintf("Thực hiện tính toán: %s", expr), "tool")
	return tools.Calculate(expr)
}

func (d *Dispatcher) handleHandoff(args handlers.Args) string {
	targetAgent, _ := args["target_agent"].(string)
	reason, _ := args["reason"].(string)
	payload, _ := args["task_payload"].(string)

	fmt.Printf("🔀 [Orchestrator] Handoff requested to %s. Reason: %s\n", targetAgent, reason)

	d.agent.mu.Lock()
	d.agent.handoffPlan = &RoutePlan{
		Agent:  targetAgent,
		Skills: guessSkillsForAgent(targetAgent),
		Reason: fmt.Sprintf("Handoff from previous agent: %s. Task: %s", reason, payload),
	}
	d.agent.mu.Unlock()

	return fmt.Sprintf("Successfully initiated handoff to %s.", targetAgent)
}

func (d *Dispatcher) handleLoadContext(args handlers.Args) string {
	docType, _ := args["type"].(string)
	docName, _ := args["name"].(string)
	pubsub.BroadcastLog(fmt.Sprintf("Nạp tài liệu: %s/%s", docType, docName), "process")
	return tools.LoadDocument(docType, docName)
}

func (d *Dispatcher) handleExportReport(args handlers.Args) string {
	format, _ := args["format"].(string)
	if format == "" {
		format, _ = args["type"].(string)
	}
	format = strings.ToLower(format)
	if format != "xlsx" && format != "pptx" {
		format = "xlsx"
	}

	title, _ := args["title"].(string)
	if title == "" {
		title = "Báo cáo Tài chính"
	}

	dataStr, _ := args["data"].(string)

	exportsDir := resolveExportsDir()
	if err := os.MkdirAll(exportsDir, 0755); err != nil {
		return fmt.Sprintf("Error creating exports directory: %v", err)
	}

	filename := fmt.Sprintf("report_%d.%s", time.Now().UnixNano(), format)
	outputPath := filepath.Join(exportsDir, filename)

	if format == "xlsx" {
		_, err := report_gen.Generate(report_gen.FormatXLSX, title, dataStr, outputPath)
		if err != nil {
			return fmt.Sprintf("Error generating Excel report: %v", err)
		}
	} else {
		outputPath = d.generatePPTXWithScript(title, dataStr, outputPath)
	}

	displayFormat := strings.ToUpper(format)
	return fmt.Sprintf("[Tải về Báo cáo (%s)](/exports/%s)", displayFormat, filename)
}

func (d *Dispatcher) handleReadLocalFile(args handlers.Args) string {
	path, _ := args["path"].(string)
	if path == "" {
		return "Error: Missing path parameter"
	}
	pubsub.BroadcastLog(fmt.Sprintf("Đang đọc tệp nội bộ: %s", path), "tool")

	baseName := filepath.Base(path)
	if baseName == "" || baseName == "." || baseName == ".." {
		return "Error: Tên tệp không hợp lệ"
	}

	allowedExts := map[string]bool{
		".xlsx": true, ".xls": true, ".csv": true, ".txt": true,
		".md": true, ".json": true, ".xml": true, ".pdf": true,
		".docx": true, ".pptx": true, ".png": true, ".jpg": true,
	}
	ext := strings.ToLower(filepath.Ext(baseName))
	if !allowedExts[ext] {
		return fmt.Sprintf("Error: Định dạng file '%s' không được phép đọc", ext)
	}

	searchDirs := []string{"frontend/exports", "Gemini/frontend/exports", "../frontend/exports", "exports"}
	var targetPath string
	for _, dir := range searchDirs {
		p := filepath.Join(dir, baseName)
		if _, err := os.Stat(p); err == nil {
			absPath, _ := filepath.Abs(p)
			absDir, _ := filepath.Abs(dir)
			if strings.HasPrefix(absPath, absDir) {
				targetPath = absPath
				break
			}
		}
	}

	if targetPath == "" {
		return fmt.Sprintf("Lỗi: Không tìm thấy tệp %s trong thư mục exports.", baseName)
	}

	data, err := os.ReadFile(targetPath)
	if err != nil {
		return fmt.Sprintf("Lỗi khi đọc tệp: %v", err)
	}

	dataB64 := base64.StdEncoding.EncodeToString(data)
	mimeType := ""

	if content, ok := utils.ParseAttachment(baseName, mimeType, dataB64); ok {
		return utils.GetFileContentWrapper(baseName, content)
	}

	return fmt.Sprintf("Thông báo: Tệp '%s' là định dạng nhị phân (Ảnh/PDF). Hãy kiểm tra file đính kèm hoặc dùng công cụ xem file.", baseName)
}

// ─── Helper Methods ──────────────────────────────────────────────────────────

func (d *Dispatcher) cacheGet(prefix, key string) (string, bool) {
	return d.cache.Get(prefix + "_" + strings.ToLower(strings.TrimSpace(key)))
}

func (d *Dispatcher) cachePut(prefix, key, value string) {
	d.cache.Put(prefix+"_"+strings.ToLower(strings.TrimSpace(key)), value)
}

// appendFunctionResponse appends a tool response to the conversation history.
func (d *Dispatcher) appendFunctionResponse(toolCall *messaging.ToolCall, result string) {
	d.agent.mu.Lock()
	defer d.agent.mu.Unlock()
	d.agent.conversation.ContextWindow.History = append(d.agent.conversation.ContextWindow.History, messaging.Message{
		Role: messaging.RoleTool,
		ToolResponses: []messaging.ToolResponse{{
			CallID:  toolCall.ID,
			Name:    toolCall.Name,
			Content: result,
		}},
	})
}

func (d *Dispatcher) generatePPTXWithScript(title, dataStr, outputPath string) string {
	scriptPath := os.Getenv("REPORT_GENERATOR_PATH")
	if scriptPath == "" {
		candidates := []string{
			"scripts/report_generator.py",
			"../scripts/report_generator.py",
			"../../scripts/report_generator.py",
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				scriptPath = c
				break
			}
		}
	}
	if scriptPath == "" {
		return ""
	}

	absScriptPath, _ := filepath.Abs(scriptPath)
	absOutputPath, _ := filepath.Abs(outputPath)

	cmdArgs := []string{absScriptPath, "--type", "pptx", "--output", absOutputPath, "--title", title}
	if dataStr != "" {
		tmpFile, err := os.CreateTemp("", "report_data_*.json")
		if err != nil {
			return ""
		}
		defer os.Remove(tmpFile.Name())
		tmpFile.Write([]byte(dataStr))
		tmpFile.Close()
		cmdArgs = append(cmdArgs, "--data-file", tmpFile.Name())
	}

	cmd := exec.Command("python", cmdArgs...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("⚠️ [PPTX] Python script error: %v. Stderr: %s\n", err, stderr.String())
		return ""
	}
	return outputPath
}

func resolveExportsDir() string {
	candidates := []string{
		"frontend/exports",
		"../../frontend/exports",
		"../frontend/exports",
		"exports",
	}
	for _, dir := range candidates {
		if _, err := os.Stat(filepath.Dir(dir)); err == nil {
			return dir
		}
	}
	return "exports"
}
