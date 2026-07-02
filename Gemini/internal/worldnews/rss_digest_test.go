package worldnews

import (
	"testing"
	"time"
)

func TestIsNewsInDigestWindow(t *testing.T) {
	since := time.Date(2026, 7, 1, 7, 0, 0, 0, vnTimezone)
	until := time.Date(2026, 7, 2, 7, 0, 0, 0, vnTimezone)

	cases := []struct {
		name string
		pub  time.Time
		want bool
	}{
		{"inside window", time.Date(2026, 7, 2, 6, 30, 0, 0, vnTimezone), true},
		{"at cutoff excluded", time.Date(2026, 7, 2, 7, 0, 0, 0, vnTimezone), false},
		{"after cutoff excluded", time.Date(2026, 7, 2, 9, 0, 0, 0, vnTimezone), false},
		{"before window excluded", time.Date(2026, 7, 1, 6, 59, 0, 0, vnTimezone), false},
		{"zero date excluded", time.Time{}, false},
	}
	for _, tc := range cases {
		got := isNewsInDigestWindow(tc.pub, since, until)
		if got != tc.want {
			t.Fatalf("%s: got %v want %v (pub=%v)", tc.name, got, tc.want, tc.pub)
		}
	}
}

func TestParsePubDateUnknownReturnsZero(t *testing.T) {
	if got := parsePubDate("not-a-date"); !got.IsZero() {
		t.Fatalf("expected zero time for unknown pubDate, got %v", got)
	}
}

func TestFilterBreakingNewsExcludesAfterCutoff(t *testing.T) {
	day := time.Date(2026, 7, 2, 0, 0, 0, 0, vnTimezone)
	since, until := morningDigestWindow(day)
	items := []rssItem{
		{Title: "Before 7h", Kind: "breaking", PubDate: time.Date(2026, 7, 2, 6, 45, 0, 0, vnTimezone)},
		{Title: "After 7h", Kind: "breaking", PubDate: time.Date(2026, 7, 2, 8, 15, 0, 0, vnTimezone)},
		{Title: "No date", Kind: "breaking", PubDate: time.Time{}},
	}
	got := filterNewsBetween(items, since, until)
	if len(got) != 1 || got[0].Title != "Before 7h" {
		t.Fatalf("expected only pre-7h breaking item, got %#v", got)
	}
}

func TestToBreakingNewsIncludesDate(t *testing.T) {
	pub := time.Date(2026, 7, 2, 6, 45, 0, 0, vnTimezone)
	got := toBreakingNews([]rssItem{
		{Title: "Fed holds rates", Source: "CNBC", PubDate: pub},
	})
	if len(got) != 1 {
		t.Fatalf("expected 1 item, got %d", len(got))
	}
	if got[0].Date != "02/07/2026" {
		t.Fatalf("expected date 02/07/2026, got %q", got[0].Date)
	}
	if got[0].Time != "06:45" {
		t.Fatalf("expected time 06:45, got %q", got[0].Time)
	}
}

func TestFilterVTVNewsExcludesAfterCutoff(t *testing.T) {
	day := time.Date(2026, 7, 2, 0, 0, 0, 0, vnTimezone)
	since, until := morningDigestWindow(day)
	items := []rssItem{
		{Title: "VTV early", Kind: "vtv", PubDate: time.Date(2026, 7, 1, 22, 0, 0, 0, vnTimezone)},
		{Title: "VTV late", Kind: "vtv", PubDate: time.Date(2026, 7, 2, 10, 0, 0, 0, vnTimezone)},
	}
	got := filterNewsBetween(items, since, until)
	if len(got) != 1 || got[0].Title != "VTV early" {
		t.Fatalf("expected only pre-cutoff VTV item, got %#v", got)
	}
}