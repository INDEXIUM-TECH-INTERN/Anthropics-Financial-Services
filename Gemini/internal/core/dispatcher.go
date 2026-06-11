package core

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gemini-cli/internal/models/messaging"
	"gemini-cli/internal/pubsub"
	"gemini-cli/internal/scripts/report_gen"
	"gemini-cli/internal/tools"
	"gemini-cli/internal/utils"
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
		d.newExportReportTool(),
		d.newReadLocalFileTool(),
	}
}

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

		result := d.resolveToolCallResult(&toolCall)
		d.appendFunctionResponse(&toolCall, result)
	}
	return true
}

func (d *Dispatcher) resolveToolCallResult(toolCall *messaging.ToolCall) string {
	switch toolCall.Name {
	case "financial_research":
		query, _ := toolCall.Args["query"].(string)
		pubsub.BroadcastLog(fmt.Sprintf("Tra cứu dữ liệu (Google): %s", query), "tool")
		return d.handleFinancialResearchTool(toolCall.Args)
	case "tavily_search":
		query, _ := toolCall.Args["query"].(string)
		pubsub.BroadcastLog(fmt.Sprintf("Tra cứu dữ liệu (Tavily): %s", query), "tool")
		return d.handleTavilySearchTool(toolCall.Args)
	case "financial_scrape":
		url, _ := toolCall.Args["url"].(string)
		pubsub.BroadcastLog(fmt.Sprintf("Đang đọc nội dung từ: %s", url), "tool")
		return d.handleFinancialScrapeTool(toolCall.Args)
	case "financial_calculate":
		expr, _ := toolCall.Args["expression"].(string)
		pubsub.BroadcastLog(fmt.Sprintf("Thực hiện tính toán: %s", expr), "tool")
		return tools.Calculate(expr)
	case "handoff_request":
		target, _ := toolCall.Args["target_agent"].(string)
		pubsub.BroadcastLog(fmt.Sprintf("Yêu cầu chuyển giao sang: %s", target), "routing")
		return d.handleHandoffTool(toolCall.Args)
	case "load_financial_context":
		docType, _ := toolCall.Args["type"].(string)
		docName, _ := toolCall.Args["name"].(string)
		pubsub.BroadcastLog(fmt.Sprintf("Nạp tài liệu: %s/%s", docType, docName), "process")
		return tools.LoadDocument(docType, docName)
	case "export_report":
		pubsub.BroadcastLog("Đang tạo và xuất báo cáo...", "tool")
		return d.handleExportReportTool(toolCall.Args)
	case "read_local_file":
		path, _ := toolCall.Args["path"].(string)
		pubsub.BroadcastLog(fmt.Sprintf("Đang đọc tệp nội bộ: %s", path), "tool")
		return d.handleReadLocalFileTool(toolCall.Args)
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
		pubsub.BroadcastLog("Dùng kết quả tìm kiếm từ cache (tiết kiệm quota)", "success")
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
		pubsub.BroadcastLog("Dùng kết quả tìm kiếm Tavily từ cache (tiết kiệm quota)", "success")
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
		pubsub.BroadcastLog("Dùng kết quả đọc nội dung từ cache (tiết kiệm quota)", "success")
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

func (d *Dispatcher) newExportReportTool() messaging.ToolSchema {
	return messaging.ToolSchema{
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
					"description": "Dữ liệu JSON thô để đưa vào báo cáo (optional). Đối với Excel có thể là danh sách các hàng hoặc dictionary chứa các sheet. Đối với PowerPoint có thể chứa bảng dữ liệu.",
				},
			},
			"required": []string{"format", "title"},
		},
	}
}

func (d *Dispatcher) handleExportReportTool(args map[string]interface{}) string {
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

	// Resolve exports directory
	exportsDir := resolveExportsDir()
	if err := os.MkdirAll(exportsDir, 0755); err != nil {
		return fmt.Sprintf("Error creating exports directory: %v", err)
	}

	// Generate filename
	filename := fmt.Sprintf("report_%d.%s", time.Now().UnixNano(), format)
	outputPath := filepath.Join(exportsDir, filename)

	// Tạo báo cáo bằng Go (Excel) hoặc Python script (PPTX)
	if format == "xlsx" {
		_, err := report_gen.Generate(report_gen.FormatXLSX, title, dataStr, outputPath)
		if err != nil {
			return fmt.Sprintf("Error generating Excel report: %v", err)
		}
	} else {
		// PPTX: dùng Python script (report_generator.py trong scripts/)
		outputPath = d.generatePPTXWithScript(title, dataStr, outputPath)
	}

	displayFormat := strings.ToUpper(format)
	return fmt.Sprintf("[Tải về Báo cáo (%s)](/exports/%s)", displayFormat, filename)
}

// generatePPTXWithScript gọi Python script để tạo PPTX
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

// resolveExportsDir tìm thư mục exports phù hợp
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


func (d *Dispatcher) newReadLocalFileTool() messaging.ToolSchema {
	return messaging.ToolSchema{
		Name:        "read_local_file",
		Description: "Đọc nội dung tệp tin cục bộ trên server (ví dụ: các tệp báo cáo đã xuất trong thư mục exports).",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{"type": "string", "description": "Tên tệp hoặc đường dẫn tệp (ví dụ: report_123.xlsx)"},
			},
			"required": []string{"path"},
		},
	}
}

func (d *Dispatcher) handleReadLocalFileTool(args map[string]interface{}) string {
	path, ok := args["path"].(string)
	if !ok {
		return "Error: Missing path parameter"
	}

	// Security: chỉ cho phép đọc file trong thư mục exports, không cho phép path traversal
	baseName := filepath.Base(path)
	if baseName == "" || baseName == "." || baseName == ".." {
		return "Error: Tên tệp không hợp lệ"
	}

	// Whitelist extensions
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
			// Verify resolved path is still inside exports directory (prevent traversal)
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

	// Đọc file và parse (R6.2)
	data, err := os.ReadFile(targetPath)
	if err != nil {
		return fmt.Sprintf("Lỗi khi đọc tệp: %v", err)
	}

	// Fake base64 để dùng chung bộ parser
	dataB64 := base64.StdEncoding.EncodeToString(data)
	mimeType := "" // Sẽ tự đoán theo ext trong ParseAttachment

	if content, ok := utils.ParseAttachment(baseName, mimeType, dataB64); ok {
		return utils.GetFileContentWrapper(baseName, content)
	}

	// Fallback nếu không parse được (ví dụ PDF, Image - gửi thông báo)
	return fmt.Sprintf("Thông báo: Tệp '%s' là định dạng nhị phân (Ảnh/PDF). Hãy kiểm tra file đính kèm hoặc dùng công cụ xem file.", baseName)
}
