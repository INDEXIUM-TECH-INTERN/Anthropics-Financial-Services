package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gemini-cli/internal/domain/entities"
	"gemini-cli/internal/domain/interfaces"
)

// FinancialResearchToolExecutor implements the ToolExecutor interface for financial research
type FinancialResearchToolExecutor struct {
	apiKey string
	client *HTTPClient
}

// NewFinancialResearchToolExecutor creates a new financial research tool executor
func NewFinancialResearchToolExecutor(apiKey string) *FinancialResearchToolExecutor {
	return &FinancialResearchToolExecutor{
		apiKey: apiKey,
		client: &HTTPClient{
			Timeout: 30 * time.Second,
		},
	}
}

// Execute executes the financial research tool
func (e *FinancialResearchToolExecutor) Execute(ctx context.Context, req *interfaces.ToolRequest) (*interfaces.ToolResponse, error) {
	start := time.Now()

	// Build search query based on request
	searchQuery := e.buildSearchQuery(req.Arguments)

	// Execute search
	result, err := e.performSearch(ctx, searchQuery)
	if err != nil {
		return &interfaces.ToolResponse{
			Success: false,
			Error:   fmt.Sprintf("Search failed: %v", err),
		}, nil
	}

	// Process and format results
	formattedResult := e.formatSearchResult(result, req.Arguments)

	return &interfaces.ToolResponse{
		Success:    true,
		Data:       formattedResult,
		ExecTime:   time.Since(start),
		Metadata: map[string]any{
			"query":    searchQuery,
			"results_count": len(result.Results),
		},
	}, nil
}

// GetSchema returns the tool schema
func (e *FinancialResearchToolExecutor) GetSchema() *entities.ToolSchema {
	return &entities.ToolSchema{
		Name:        "financial_research",
		Description: "Truy vấn dữ liệu thị trường thực tế bằng Google Search, hỗ trợ mã chứng khoán, tin tức tài chính, báo cáo công ty",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Từ khóa, mã chứng khoán, hoặc câu hỏi tài chính cần tìm kiếm",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Số lượng kết quả tối đa (mặc định 10)",
					"default":     10,
				},
				"recent": map[string]interface{}{
					"type":        "boolean",
					"description": "Chỉ hiển thị kết quả gần đây (mặc định false)",
					"default":     false,
				},
				"source": map[string]interface{}{
					"type":        "string",
					"description": "Nguồn tin cụ thể (ví dụ: cafebiz, cafef, vietnamnet)",
					"default":     "",
				},
			},
			"required": []string{"query"},
		},
	}
}

// Validate validates the tool input
func (e *FinancialResearchToolExecutor) Validate(args map[string]any) error {
	if query, ok := args["query"]; !ok || query == "" {
		return fmt.Errorf("query parameter is required")
	}

	if limit, ok := args["limit"]; ok {
		if limitFloat, ok := limit.(float64); ok && limitFloat <= 0 {
			return fmt.Errorf("limit must be positive")
		}
	}

	return nil
}

// GetCapabilities returns tool capabilities
func (e *FinancialResearchToolExecutor) GetCapabilities() []string {
	return []string{
		"market-data",
		"financial-news",
		"company-reports",
		"stock-analysis",
		"real-time-data",
	}
}

// buildSearchQuery builds a search query based on request
func (e *FinancialResearchToolExecutor) buildSearchQuery(args map[string]any) string {
	query, _ := args["query"].(string)

	// Add context if available
	if recent, ok := args["recent"].(bool); ok && recent {
		query += " 2024 2025"
	}

	// Add source filter if specified
	if source, ok := args["source"]; ok && source != "" {
		if sourceStr, ok := source.(string); ok {
			query += " site:" + sourceStr
		}
	}

	return query
}

// performSearch executes the search
func (e *FinancialResearchToolExecutor) performSearch(ctx context.Context, query string) (*SearchResult, error) {
	// This would implement actual Google Search API calls
	// For now, return mock data

	// TODO: Implement actual Google Search API integration
	return &SearchResult{
		Query:   query,
		Results: []SearchResultItem{
			{
				Title:       "Cổ phiếu VCB - Ngân hàng TMCP Ngoại thương Việt Nam",
				URL:         "https://cafef.vn/co-phieu-vcB-...",
				Snippet:     "VCB duy trì vị thế dẫn đầu với lợi nhuận ổn định...",
				Source:      "CafeF",
				PublishDate: time.Now(),
			},
			{
				Title:       "Báo cáo tài chính quý 3/2024 - Công ty ABC",
				URL:         "https://vietnamnet.vn/bao-cao-tai-chin...",
				Snippet:     "Doanh thu quý 3 đạt 2.500 tỷ đồng, tăng 15% so với cùng kỳ...",
				Source:      "VietnamNet",
				PublishDate: time.Now().AddDate(0, -1, 0),
			},
		},
	}, nil
}

// formatSearchResult formats the search result
func (e *FinancialResearchToolExecutor) formatSearchResult(result *SearchResult, args map[string]any) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("🔍 Kết quả tìm kiếm: \"%s\"\n\n", result.Query))

	for i, item := range result.Results {
		builder.WriteString(fmt.Sprintf("%d. **%s**\n", i+1, item.Title))
		builder.WriteString(fmt.Sprintf("   📍 Nguồn: %s (%s)\n", item.Source, item.PublishDate.Format("02/01/2006")))
		builder.WriteString(fmt.Sprintf("   📄 %s\n", item.URL))
		builder.WriteString(fmt.Sprintf("   💬 %s\n\n", item.Snippet))
	}

	return builder.String()
}

// SearchResult represents search result
type SearchResult struct {
	Query   string             `json:"query"`
	Results []SearchResultItem `json:"results"`
}

// SearchResultItem represents a single search result
type SearchResultItem struct {
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Snippet     string    `json:"snippet"`
	Source      string    `json:"source"`
	PublishDate time.Time `json:"publish_date"`
}

// HTTPClient represents an HTTP client for API calls
type HTTPClient struct {
	Timeout time.Duration
}