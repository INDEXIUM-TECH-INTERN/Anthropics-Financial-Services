package worldnews

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type feedSource struct {
	Name string
	URL  string
	Kind string // global, vietnam, breaking, vtv
}

// stockFeeds — nguồn duy nhất cho mục Chứng khoán Thế giới (CNBC, WSJ, Reuters).
var stockFeeds = []feedSource{
	{Name: "CNBC", URL: "https://search.cnbc.com/rs/search/combinedcms/view.xml?partnerId=wrss01&id=100003114", Kind: "stocks"},
	{
		Name: "Reuters",
		URL:  "https://news.google.com/rss/search?q=site:reuters.com+markets+stocks+when:2d&hl=en-US&gl=US&ceid=US:en",
		Kind: "stocks",
	},
	{
		Name: "WSJ",
		URL:  "https://news.google.com/rss/search?q=site:wsj.com+finance+stocks+when:2d&hl=en-US&gl=US&ceid=US:en",
		Kind: "stocks",
	},
}

var globalFeeds = []feedSource{
	{Name: "CNBC", URL: "https://search.cnbc.com/rs/search/combinedcms/view.xml?partnerId=wrss01&id=100003114", Kind: "global"},
	{Name: "Bloomberg", URL: "https://feeds.bloomberg.com/markets/news.rss", Kind: "breaking"},
	{Name: "Financial Times", URL: "https://www.ft.com/rss/home", Kind: "breaking"},
	{Name: "CNN Business", URL: "http://rss.cnn.com/rss/money_latest.rss", Kind: "breaking"},
}

var vietnamFeeds = []feedSource{
	{Name: "VNeconomy", URL: "https://vneconomy.vn/rss/home.rss", Kind: "vietnam"},
	{Name: "VNExpress Kinh doanh", URL: "https://vnexpress.net/rss/kinh-doanh.rss", Kind: "vietnam"},
	{Name: "Vietstock", URL: "https://vietstock.vn/rss", Kind: "vietnam"},
	{Name: "Diễn đàn Doanh nghiệp", URL: "https://diendandoanhnghiep.vn/feed", Kind: "vietnam"},
	{Name: "Nhịp cầu đầu tư", URL: "https://nhipcaudautu.vn/feed/", Kind: "vietnam"},
}

// VTVIndex chưa có RSS công khai — dùng Google News làm nguồn thay thế tạm thời.
var vtvFeed = feedSource{
	Name: "VTVIndex Thế giới",
	URL:  "https://news.google.com/rss/search?q=site:vtv.vn+kinh+te+OR+site:vtvindex.vn&hl=vi&gl=VN&ceid=VN:vi",
	Kind: "vtv",
}

type rssItem struct {
	Title         string
	Link          string
	Description   string
	PubDate       time.Time
	Source        string
	Kind          string
	Thumbnail     string
	PublisherHost string
	RawXML        string
}

var itemBlockRe = regexp.MustCompile(`(?is)<item\b[^>]*>(.*?)</item>`)
var atomEntryRe = regexp.MustCompile(`(?is)<entry\b[^>]*>(.*?)</entry>`)

type rss2Channel struct {
	Items []rss2Item `xml:"channel>item"`
}

type rss2Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

type atomFeed struct {
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	Title   string    `xml:"title"`
	Link    atomLink  `xml:"link"`
	Summary string    `xml:"summary"`
	Updated string    `xml:"updated"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
}

var tagRe = regexp.MustCompile(`<[^>]+>`)

func isAllowedStockNewsSource(it rssItem) bool {
	allowedHosts := map[string]struct{}{
		"cnbc.com":   {},
		"reuters.com": {},
		"wsj.com":    {},
	}
	if _, ok := allowedHosts[it.PublisherHost]; ok {
		return true
	}
	source := strings.ToLower(strings.TrimSpace(it.Source))
	switch {
	case source == "cnbc", source == "reuters", source == "wsj":
		return true
	case strings.Contains(source, "wall street journal"):
		return true
	default:
		return false
	}
}

func filterStockNewsSources(items []rssItem) []rssItem {
	var out []rssItem
	for _, it := range items {
		if isAllowedStockNewsSource(it) {
			out = append(out, it)
		}
	}
	return out
}

func (s *Service) fetchStockNewsForReport(calendarDay time.Time) []rssItem {
	since, until := morningDigestWindow(calendarDay)

	var items []rssItem
	for _, src := range stockFeeds {
		feedItems, err := s.fetchFeed(src)
		if err != nil {
			continue
		}
		items = append(items, feedItems...)
	}

	items = filterNewsBetween(items, since, until)
	if len(items) < 2 {
		historical := s.fetchHistoricalStockGoogleNews(since, until)
		items = dedupeNews(append(items, historical...))
	}

	return s.enrichItemMedia(filterStockNewsSources(dedupeNews(items)), 4)
}

func (s *Service) fetchHistoricalStockGoogleNews(since, until time.Time) []rssItem {
	after := since.Format("2006-01-02")
	before := until.Format("2006-01-02")
	query := fmt.Sprintf(
		"(site:cnbc.com/world OR site:wsj.com/finance/stocks OR site:reuters.com/markets/stocks) after:%s before:%s",
		after, before,
	)
	src := feedSource{
		Name: "Google News (Chứng khoán)",
		URL:  fmt.Sprintf("https://news.google.com/rss/search?q=%s&hl=en-US&gl=US&ceid=US:en", url.QueryEscape(query)),
		Kind: "stocks",
	}
	items, err := s.fetchFeed(src)
	if err != nil {
		return nil
	}
	return filterNewsBetween(items, since, until)
}

func (s *Service) fetchNewsForReport(calendarDay time.Time) []rssItem {
	since, until := morningDigestWindow(calendarDay)
	var items []rssItem
	for _, src := range vietnamFeeds {
		feedItems, err := s.fetchFeed(src)
		if err != nil {
			continue
		}
		items = append(items, feedItems...)
	}
	filtered := filterNewsBetween(items, since, until)
	if len(filtered) < 4 {
		historical := s.fetchHistoricalGoogleNews(since, until)
		filtered = filterNewsBetween(dedupeNews(append(filtered, historical...)), since, until)
	}
	return s.enrichItemMedia(filtered, 6)
}

// fetchBreakingNewsForReport returns breaking/global headlines published in the 24h window
// ending at 07:00 GMT+7 on calendarDay (exclusive — nothing at or after 07:00).
func (s *Service) fetchBreakingNewsForReport(calendarDay time.Time) []rssItem {
	since, until := morningDigestWindow(calendarDay)
	var items []rssItem
	for _, src := range globalFeeds {
		if src.Kind != "breaking" && src.Name != "CNBC" {
			continue
		}
		feedItems, err := s.fetchFeed(src)
		if err != nil {
			continue
		}
		items = append(items, feedItems...)
	}
	items = filterNewsBetween(items, since, until)
	if len(items) < 3 {
		historical := s.fetchHistoricalBreakingNews(since, until)
		items = filterNewsBetween(dedupeNews(append(items, historical...)), since, until)
	}
	return s.enrichItemMedia(sortNewsByDateDesc(items), 8)
}

// fetchVTVNewsForReport returns VTV / VTVIndex articles before 07:00 GMT+7 on calendarDay.
func (s *Service) fetchVTVNewsForReport(calendarDay time.Time) []rssItem {
	since, until := morningDigestWindow(calendarDay)
	items, err := s.fetchFeed(vtvFeed)
	if err != nil {
		items = nil
	}
	items = filterNewsBetween(items, since, until)
	if len(items) < 2 {
		historical := s.fetchHistoricalVTVNews(since, until)
		items = filterNewsBetween(dedupeNews(append(items, historical...)), since, until)
	}
	return s.enrichItemMedia(sortNewsByDateDesc(items), 5)
}

type historicalNewsQuery struct {
	query string
	name  string
}

func (s *Service) fetchHistoricalBreakingNews(since, until time.Time) []rssItem {
	after := since.Format("2006-01-02")
	before := until.Format("2006-01-02")
	queries := []historicalNewsQuery{
		{
			query: fmt.Sprintf("(breaking OR \"stock market\" OR Fed OR oil OR gold) after:%s before:%s", after, before),
			name:  "Google News (Breaking)",
		},
		{
			query: fmt.Sprintf("(site:cnbc.com OR site:bloomberg.com OR site:ft.com) after:%s before:%s", after, before),
			name:  "Google News (Breaking CNBC/Bloomberg/FT)",
		},
	}
	return s.fetchHistoricalNewsQueries(queries, "breaking", since, until)
}

func (s *Service) fetchHistoricalVTVNews(since, until time.Time) []rssItem {
	after := since.Format("2006-01-02")
	before := until.Format("2006-01-02")
	queries := []historicalNewsQuery{
		{
			query: fmt.Sprintf("(site:vtv.vn OR site:vtvindex.vn) (kinh te OR tai chinh) after:%s before:%s", after, before),
			name:  "Google News (VTV)",
		},
	}
	return s.fetchHistoricalNewsQueries(queries, "vtv", since, until)
}

func (s *Service) fetchHistoricalNewsQueries(
	queries []historicalNewsQuery,
	kind string,
	since, until time.Time,
) []rssItem {
	var out []rssItem
	for _, q := range queries {
		src := feedSource{
			Name: q.name,
			URL:  fmt.Sprintf("https://news.google.com/rss/search?q=%s&hl=vi&gl=VN&ceid=VN:vi", url.QueryEscape(q.query)),
			Kind: kind,
		}
		items, err := s.fetchFeed(src)
		if err != nil {
			continue
		}
		out = append(out, filterNewsBetween(items, since, until)...)
	}
	return dedupeNews(out)
}

func takeNewsItems(items []rssItem, limit int) []rssItem {
	if limit <= 0 || len(items) <= limit {
		return items
	}
	return items[:limit]
}

func (s *Service) fetchHistoricalGoogleNews(since, until time.Time) []rssItem {
	after := since.Format("2006-01-02")
	before := until.Format("2006-01-02")

	queries := []struct {
		query  string
		name   string
		kind   string
	}{
		{
			query: fmt.Sprintf("(stock market OR S&P 500 OR Nasdaq) after:%s before:%s", after, before),
			name:  "Google News (Thế giới)",
			kind:  "breaking",
		},
		{
			query: fmt.Sprintf("(oil OR crude OR gold OR USD) after:%s before:%s", after, before),
			name:  "Google News (Hàng hóa)",
			kind:  "global",
		},
		{
			query: fmt.Sprintf("(kinh te OR tai chinh) after:%s before:%s", after, before),
			name:  "Google News (Việt Nam)",
			kind:  "vietnam",
		},
	}

	var out []rssItem
	for _, q := range queries {
		src := feedSource{
			Name: q.name,
			URL:  fmt.Sprintf("https://news.google.com/rss/search?q=%s&hl=vi&gl=VN&ceid=VN:vi", url.QueryEscape(q.query)),
			Kind: q.kind,
		}
		items, err := s.fetchFeed(src)
		if err != nil {
			continue
		}
		out = append(out, filterNewsBetween(items, since, until)...)
	}
	return dedupeNews(out)
}

func filterNewsBetween(items []rssItem, since, until time.Time) []rssItem {
	since = since.In(vnTimezone)
	until = until.In(vnTimezone)
	var out []rssItem
	for _, it := range items {
		if !isNewsInDigestWindow(it.PubDate, since, until) {
			continue
		}
		out = append(out, it)
	}
	return out
}

// isNewsInDigestWindow keeps articles with since <= pubDate < until (07:00 GMT+7 cutoff).
func isNewsInDigestWindow(pubDate, since, until time.Time) bool {
	if pubDate.IsZero() {
		return false
	}
	pub := pubDate.In(vnTimezone)
	return !pub.Before(since) && pub.Before(until)
}

func sortNewsByDateDesc(items []rssItem) []rssItem {
	if len(items) < 2 {
		return items
	}
	sorted := append([]rssItem(nil), items...)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].PubDate.After(sorted[i].PubDate) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	return sorted
}

func dedupeNews(items []rssItem) []rssItem {
	seen := make(map[string]struct{})
	var out []rssItem
	for _, it := range items {
		key := strings.ToLower(strings.TrimSpace(it.Title))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, it)
	}
	return out
}

func (s *Service) fetchAllNews() ([]rssItem, error) {
	sources := append(append([]feedSource{}, globalFeeds...), vietnamFeeds...)
	sources = append(sources, vtvFeed)

	var all []rssItem
	for _, src := range sources {
		items, err := s.fetchFeed(src)
		if err != nil {
			continue
		}
		all = append(all, items...)
	}
	return all, nil
}

func (s *Service) fetchFeed(src feedSource) ([]rssItem, error) {
	body, err := s.httpGet(src.URL)
	if err != nil {
		return nil, err
	}
	return parseFeedBody(body, src), nil
}

func parseFeedBody(body []byte, src feedSource) []rssItem {
	raw := string(body)
	if blocks := itemBlockRe.FindAllString(raw, -1); len(blocks) > 0 {
		var rss rss2Channel
		if err := xml.Unmarshal(body, &rss); err == nil && len(rss.Items) > 0 {
			return mapRSS2Items(rss.Items, src, blocks)
		}
	}
	if blocks := atomEntryRe.FindAllString(raw, -1); len(blocks) > 0 {
		var atom atomFeed
		if err := xml.Unmarshal(body, &atom); err == nil && len(atom.Entries) > 0 {
			return mapAtomEntries(atom.Entries, src, blocks)
		}
	}
	return nil
}

func mapRSS2Items(items []rss2Item, src feedSource, rawBlocks []string) []rssItem {
	out := make([]rssItem, 0, len(items))
	for i, it := range items {
		title := cleanText(it.Title)
		if title == "" {
			continue
		}
		raw := ""
		if i < len(rawBlocks) {
			raw = rawBlocks[i]
		}
		sourceName := src.Name
		if pub := publisherFromGoogleTitle(title); pub != "" {
			sourceName = pub
		}
		link := strings.TrimSpace(it.Link)
		host := extractPublisherHost(raw, link, sourceName)
		thumb := extractThumbnailFromItemXML(raw)
		out = append(out, rssItem{
			Title:         title,
			Link:          link,
			Description:   cleanText(it.Description),
			PubDate:       parsePubDate(it.PubDate),
			Source:        sourceName,
			Kind:          src.Kind,
			Thumbnail:     thumb,
			PublisherHost: host,
			RawXML:        raw,
		})
	}
	return out
}

func mapAtomEntries(entries []atomEntry, src feedSource, rawBlocks []string) []rssItem {
	out := make([]rssItem, 0, len(entries))
	for i, e := range entries {
		title := cleanText(e.Title)
		if title == "" {
			continue
		}
		pub := parsePubDate(e.Updated)
		raw := ""
		if i < len(rawBlocks) {
			raw = rawBlocks[i]
		}
		link := strings.TrimSpace(e.Link.Href)
		host := extractPublisherHost(raw, link, src.Name)
		thumb := extractThumbnailFromItemXML(raw)
		out = append(out, rssItem{
			Title:         title,
			Link:          link,
			Description:   cleanText(e.Summary),
			PubDate:       pub,
			Source:        src.Name,
			Kind:          src.Kind,
			Thumbnail:     thumb,
			PublisherHost: host,
			RawXML:        raw,
		})
	}
	return out
}

func publisherFromGoogleTitle(title string) string {
	if idx := strings.LastIndex(title, " - "); idx >= 0 {
		return strings.TrimSpace(title[idx+3:])
	}
	return ""
}

func cleanText(s string) string {
	s = tagRe.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	return strings.TrimSpace(s)
}

func parsePubDate(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	formats := []string{
		time.RFC1123,
		time.RFC1123Z,
		time.RFC822,
		time.RFC822Z,
		time.RFC3339,
		"Mon, 02 Jan 2006 15:04:05 MST",
		"2006-01-02T15:04:05Z",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, raw); err == nil {
			return t
		}
	}
	return time.Time{}
}

func filterRecent(items []rssItem, since time.Time, limit int) []rssItem {
	var out []rssItem
	for _, it := range items {
		if it.PubDate.Before(since) {
			continue
		}
		out = append(out, it)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func toNewsArticles(items []rssItem) []NewsArticle {
	out := make([]NewsArticle, 0, len(items))
	for _, it := range items {
		summary := it.Description
		if len(summary) > 220 {
			summary = summary[:217] + "..."
		}
		timeLabel := it.PubDate.In(vnTimezone).Format("02/01/2006 15:04")
		out = append(out, NewsArticle{
			Title:     it.Title,
			Summary:   summary,
			Source:    it.Source,
			Time:      timeLabel,
			URL:       it.Link,
			Thumbnail: mediaField(it.Thumbnail, true),
			Logo:      mediaField(it.PublisherHost, false),
		})
	}
	return out
}

func toBreakingNews(items []rssItem) []BreakingNews {
	out := make([]BreakingNews, 0, len(items))
	for _, it := range items {
		urgent := strings.Contains(strings.ToLower(it.Title), "breaking") ||
			strings.Contains(strings.ToLower(it.Title), "fed") ||
			strings.Contains(strings.ToLower(it.Title), "oil") ||
			strings.Contains(strings.ToLower(it.Title), "war")
		out = append(out, BreakingNews{
			Time:      it.PubDate.In(vnTimezone).Format("15:04"),
			Source:    it.Source,
			Content:   it.Title,
			URL:       it.Link,
			Thumbnail: mediaField(it.Thumbnail, true),
			Logo:      mediaField(it.PublisherHost, false),
			IsUrgent:  urgent,
		})
	}
	return out
}

func relativeTime(t time.Time) string {
	diff := time.Since(t)
	switch {
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins < 1 {
			mins = 1
		}
		return fmtDuration(mins, "phút")
	case diff < 24*time.Hour:
		return fmtDuration(int(diff.Hours()), "giờ")
	default:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 ngày trước"
		}
		return fmt.Sprintf("%d ngày trước", days)
	}
}

func fmtDuration(n int, unit string) string {
	return fmt.Sprintf("%d %s trước", n, unit)
}

func mediaField(value string, isImage bool) string {
	if value == "" {
		return ""
	}
	if isImage {
		return ImageProxyPath(value)
	}
	return FaviconProxyPath(value)
}

