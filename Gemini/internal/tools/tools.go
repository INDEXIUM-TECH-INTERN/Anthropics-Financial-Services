package tools

import (
	"fmt"
	"time"

	"gemini-cli/internal/tools/market"
	"gemini-cli/internal/tools/registry"
	"gemini-cli/internal/tools/scraper"
	"gemini-cli/internal/utils"
)

type MarketQueryPlan = market.MarketQueryPlan
type LoadedDocument = registry.LoadedDocument

func SearchGoogle(query string) string {
	return market.SearchGoogle(query)
}

func ScrapeWeb(targetURL string) string {
	return scraper.ScrapeWeb(targetURL)
}

func Calculate(expression string) string {
	fmt.Printf("🧮 [Tool] Đang tính toán: %s...\n", expression)
	return fmt.Sprintf("Kết quả tính toán cho '%s' (Mô phỏng): Dữ liệu đã được xử lý.", expression)
}

func NeedsRealtimeData(query string) bool {
	return market.NeedsRealtimeData(query)
}

func BuildMarketQueryPlan(query string) MarketQueryPlan {
	return market.BuildMarketQueryPlan(query)
}

func GetCurrentTime() string {
	now := time.Now()
	return fmt.Sprintf("Ngày hiện tại: %s, %s (Múi giờ: Asia/Ho_Chi_Minh)",
		now.Format("02/01/2006 15:04:05"),
		utils.TranslateWeekday(now.Weekday()))
}

func LoadDocument(docType, name string) string {
	return registry.LoadDocument(docType, name)
}

func LoadDocumentWithMetadata(docType, name string) LoadedDocument {
	return registry.LoadDocumentWithMetadata(docType, name)
}

func GetRoutingGuide() string {
	return registry.GetRoutingGuide()
}
