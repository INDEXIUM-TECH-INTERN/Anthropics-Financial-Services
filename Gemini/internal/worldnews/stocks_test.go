package worldnews

import "testing"

func TestIsAllowedStockNewsSource(t *testing.T) {
	cases := []struct {
		item rssItem
		want bool
	}{
		{rssItem{Source: "CNBC", PublisherHost: "cnbc.com"}, true},
		{rssItem{Source: "Reuters", PublisherHost: "reuters.com"}, true},
		{rssItem{Source: "WSJ", PublisherHost: "wsj.com"}, true},
		{rssItem{Source: "Wall Street Journal", PublisherHost: "wsj.com"}, true},
		{rssItem{Source: "Bloomberg", PublisherHost: "bloomberg.com"}, false},
		{rssItem{Source: "Google News (Thế giới)", PublisherHost: "news.google.com"}, false},
	}

	for _, tc := range cases {
		got := isAllowedStockNewsSource(tc.item)
		if got != tc.want {
			t.Fatalf("source %q host %q: got %v want %v", tc.item.Source, tc.item.PublisherHost, got, tc.want)
		}
	}
}

func TestFilterStockNewsSources(t *testing.T) {
	items := []rssItem{
		{Title: "A", Source: "CNBC", PublisherHost: "cnbc.com"},
		{Title: "B", Source: "Bloomberg", PublisherHost: "bloomberg.com"},
		{Title: "C", Source: "Reuters", PublisherHost: "reuters.com"},
	}
	filtered := filterStockNewsSources(items)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 stock items, got %d", len(filtered))
	}
}