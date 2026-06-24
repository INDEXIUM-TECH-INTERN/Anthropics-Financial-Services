package worldnews

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type feedSource struct {
	Name string
	URL  string
	Kind string // global, vietnam, breaking, vtv
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
	Title       string
	Link        string
	Description string
	PubDate     time.Time
	Source      string
	Kind        string
}

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
	var rss rss2Channel
	if err := xml.Unmarshal(body, &rss); err == nil && len(rss.Items) > 0 {
		return mapRSS2Items(rss.Items, src)
	}

	var atom atomFeed
	if err := xml.Unmarshal(body, &atom); err == nil && len(atom.Entries) > 0 {
		return mapAtomEntries(atom.Entries, src)
	}
	return nil
}

func mapRSS2Items(items []rss2Item, src feedSource) []rssItem {
	out := make([]rssItem, 0, len(items))
	for _, it := range items {
		title := cleanText(it.Title)
		if title == "" {
			continue
		}
		out = append(out, rssItem{
			Title:       title,
			Link:        strings.TrimSpace(it.Link),
			Description: cleanText(it.Description),
			PubDate:     parsePubDate(it.PubDate),
			Source:      src.Name,
			Kind:        src.Kind,
		})
	}
	return out
}

func mapAtomEntries(entries []atomEntry, src feedSource) []rssItem {
	out := make([]rssItem, 0, len(entries))
	for _, e := range entries {
		title := cleanText(e.Title)
		if title == "" {
			continue
		}
		pub := parsePubDate(e.Updated)
		out = append(out, rssItem{
			Title:       title,
			Link:        strings.TrimSpace(e.Link.Href),
			Description: cleanText(e.Summary),
			PubDate:     pub,
			Source:      src.Name,
			Kind:        src.Kind,
		})
	}
	return out
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
	return time.Now()
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
		out = append(out, NewsArticle{
			Title:   it.Title,
			Summary: summary,
			Source:  it.Source,
			Time:    relativeTime(it.PubDate),
			URL:     it.Link,
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
			Time:     it.PubDate.In(vnTimezone).Format("15:04"),
			Source:   it.Source,
			Content:  it.Title,
			IsUrgent: urgent,
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

