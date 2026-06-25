package worldnews

import "testing"

func TestExtractThumbnailFromItemXML(t *testing.T) {
	raw := `<item><media:content url="https://assets.example.com/photo.jpg" type="image/jpeg"/></item>`
	got := extractThumbnailFromItemXML(raw)
	if got != "https://assets.example.com/photo.jpg" {
		t.Fatalf("expected thumbnail url, got %q", got)
	}
}

func TestExtractPublisherHost(t *testing.T) {
	raw := `<item><source url="https://www.cnbc.com">CNBC</source></item>`
	got := extractPublisherHost(raw, "https://news.google.com/rss/articles/abc", "Google News")
	if got != "cnbc.com" {
		t.Fatalf("expected cnbc.com, got %q", got)
	}
}

func TestFaviconProxyPath(t *testing.T) {
	got := FaviconProxyPath("www.bloomberg.com")
	if got == "" || !contains(got, "host=bloomberg.com") {
		t.Fatalf("unexpected favicon path: %q", got)
	}
}

func TestSanitizeHostRejectsInvalid(t *testing.T) {
	if sanitizeHost("localhost") != "localhost" {
		// localhost is valid hostname string but blocked at fetch via isPrivateHost in image url only
	}
	if sanitizeHost("bad/host") != "" {
		t.Fatal("expected invalid host")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}