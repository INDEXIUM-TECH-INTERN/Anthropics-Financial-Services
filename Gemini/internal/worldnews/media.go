package worldnews

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	imgSrcRe         = regexp.MustCompile(`(?i)<img[^>]+src=["']([^"']+)["']`)
	mediaContentRe   = regexp.MustCompile(`(?i)<media:content[^>]+url=["']([^"']+)["']`)
	mediaThumbnailRe = regexp.MustCompile(`(?i)<media:thumbnail[^>]+url=["']([^"']+)["']`)
	enclosureImgRe   = regexp.MustCompile(`(?i)<enclosure[^>]+url=["']([^"']+)["'][^>]*type=["']image[^"']*["']`)
	enclosureImgRe2  = regexp.MustCompile(`(?i)<enclosure[^>]+type=["']image[^"']*["'][^>]+url=["']([^"']+)["']`)
	sourceURLRe      = regexp.MustCompile(`(?i)<source[^>]+url=["']([^"']+)["']`)
	ogImageRe        = regexp.MustCompile(`(?i)<meta[^>]+property=["']og:image(?::secure_url)?["'][^>]+content=["']([^"']+)["']`)
	ogImageRe2       = regexp.MustCompile(`(?i)<meta[^>]+content=["']([^"']+)["'][^>]+property=["']og:image(?::secure_url)?["']`)
	twitterImageRe   = regexp.MustCompile(`(?i)<meta[^>]+name=["']twitter:image["'][^>]+content=["']([^"']+)["']`)

	knownSourceHosts = map[string]string{
		"CNBC":                    "cnbc.com",
		"Bloomberg":               "bloomberg.com",
		"Financial Times":         "ft.com",
		"CNN Business":            "cnn.com",
		"VNeconomy":               "vneconomy.vn",
		"VNExpress Kinh doanh":    "vnexpress.net",
		"Vietstock":               "vietstock.vn",
		"Diễn đàn Doanh nghiệp":   "diendandoanhnghiep.vn",
		"Nhịp cầu đầu tư":         "nhipcaudautu.vn",
		"VTVIndex Thế giới":       "vtv.vn",
		"Google News (Thế giới)":    "news.google.com",
		"Google News (Hàng hóa)":    "news.google.com",
		"Google News (Việt Nam)":    "news.google.com",
		"Google News (Chứng khoán)": "news.google.com",
		"Reuters":                 "reuters.com",
		"WSJ":                     "wsj.com",
		"Wall Street Journal":     "wsj.com",
	}
)

type mediaCache struct {
	mu    sync.RWMutex
	items map[string]cacheMediaEntry
}

type cacheMediaEntry struct {
	data      []byte
	mime      string
	expiresAt time.Time
}

var faviconCache = mediaCache{items: make(map[string]cacheMediaEntry)}
var imageCache = mediaCache{items: make(map[string]cacheMediaEntry)}

func extractThumbnailFromItemXML(raw string) string {
	for _, re := range []*regexp.Regexp{mediaThumbnailRe, mediaContentRe, enclosureImgRe, enclosureImgRe2, imgSrcRe} {
		if m := re.FindStringSubmatch(raw); len(m) > 1 {
			if u := normalizeImageURL(m[1]); u != "" {
				return u
			}
		}
	}
	return ""
}

func extractPublisherHost(raw, link, sourceName string) string {
	if m := sourceURLRe.FindStringSubmatch(raw); len(m) > 1 {
		if host := hostFromURL(m[1]); host != "" {
			return host
		}
	}
	if host := hostFromURL(link); host != "" && !isAggregatorHost(host) {
		return host
	}
	if host, ok := knownSourceHosts[sourceName]; ok {
		return host
	}
	return hostFromSourceLabel(sourceName)
}

func hostFromSourceLabel(label string) string {
	label = strings.TrimSpace(label)
	if host, ok := knownSourceHosts[label]; ok {
		return host
	}
	// "Title - WSJ" in Google News item titles handled separately in mapRSS2
	parts := strings.Split(label, " - ")
	if len(parts) >= 2 {
		pub := strings.TrimSpace(parts[len(parts)-1])
		if host, ok := knownSourceHosts[pub]; ok {
			return host
		}
		lower := strings.ToLower(pub)
		for name, host := range knownSourceHosts {
			if strings.EqualFold(name, pub) || strings.Contains(strings.ToLower(name), lower) {
				return host
			}
		}
	}
	return ""
}

func hostFromURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	if idx := strings.Index(host, ":"); idx >= 0 {
		host = host[:idx]
	}
	return strings.TrimPrefix(host, "www.")
}

func isAggregatorHost(host string) bool {
	switch host {
	case "news.google.com", "google.com", "news.google.com.vn":
		return true
	default:
		return false
	}
}

func normalizeImageURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "//") {
		raw = "https:" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return ""
	}
	if isPrivateHost(u.Hostname()) {
		return ""
	}
	return u.String()
}

func isPrivateHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
}

func FaviconProxyPath(host string) string {
	host = sanitizeHost(host)
	if host == "" {
		return ""
	}
	return fmt.Sprintf("/api/world-news/favicon?host=%s", url.QueryEscape(host))
}

func ImageProxyPath(imageURL string) string {
	imageURL = normalizeImageURL(imageURL)
	if imageURL == "" {
		return ""
	}
	return fmt.Sprintf("/api/world-news/image?url=%s", url.QueryEscape(imageURL))
}

func sanitizeHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	host = strings.TrimPrefix(host, "www.")
	if host == "" || strings.Contains(host, "/") || strings.Contains(host, ":") {
		return ""
	}
	for _, r := range host {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '-' {
			continue
		}
		return ""
	}
	return host
}

func (s *Service) FetchFavicon(host string) ([]byte, string, error) {
	host = sanitizeHost(host)
	if host == "" {
		return nil, "", fmt.Errorf("invalid host")
	}

	faviconCache.mu.RLock()
	if entry, ok := faviconCache.items[host]; ok && time.Now().Before(entry.expiresAt) {
		faviconCache.mu.RUnlock()
		return entry.data, entry.mime, nil
	}
	faviconCache.mu.RUnlock()

	fetchURL := fmt.Sprintf("https://www.google.com/s2/favicons?domain=%s&sz=64", url.QueryEscape(host))
	body, mime, err := s.fetchBinary(fetchURL, 256*1024)
	if err != nil {
		return nil, "", err
	}
	if mime == "" {
		mime = "image/png"
	}

	faviconCache.mu.Lock()
	faviconCache.items[host] = cacheMediaEntry{data: body, mime: mime, expiresAt: time.Now().Add(24 * time.Hour)}
	faviconCache.mu.Unlock()
	return body, mime, nil
}

func (s *Service) FetchProxiedImage(imageURL string) ([]byte, string, error) {
	imageURL = normalizeImageURL(imageURL)
	if imageURL == "" {
		return nil, "", fmt.Errorf("invalid image url")
	}

	imageCache.mu.RLock()
	if entry, ok := imageCache.items[imageURL]; ok && time.Now().Before(entry.expiresAt) {
		imageCache.mu.RUnlock()
		return entry.data, entry.mime, nil
	}
	imageCache.mu.RUnlock()

	body, mime, err := s.fetchBinary(imageURL, 5*1024*1024)
	if err != nil {
		return nil, "", err
	}
	if mime == "" || !strings.HasPrefix(mime, "image/") {
		mime = http.DetectContentType(body)
	}
	if !strings.HasPrefix(mime, "image/") {
		return nil, "", fmt.Errorf("not an image")
	}

	imageCache.mu.Lock()
	imageCache.items[imageURL] = cacheMediaEntry{data: body, mime: mime, expiresAt: time.Now().Add(time.Hour)}
	imageCache.mu.Unlock()
	return body, mime, nil
}

func (s *Service) fetchBinary(fetchURL string, maxBytes int64) ([]byte, string, error) {
	req, err := http.NewRequest(http.MethodGet, fetchURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "image/*,*/*;q=0.8")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes))
	if err != nil {
		return nil, "", err
	}
	return body, resp.Header.Get("Content-Type"), nil
}

func (s *Service) fetchOGImage(pageURL string) string {
	host := hostFromURL(pageURL)
	if host == "" || isAggregatorHost(host) {
		return ""
	}
	req, err := http.NewRequest(http.MethodGet, pageURL, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := s.client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	buf, err := io.ReadAll(io.LimitReader(resp.Body, 96*1024))
	if err != nil {
		return ""
	}
	html := string(buf)
	for _, re := range []*regexp.Regexp{ogImageRe, ogImageRe2, twitterImageRe} {
		if m := re.FindStringSubmatch(html); len(m) > 1 {
			if u := normalizeImageURL(htmlUnescapeAttr(m[1])); u != "" {
				return u
			}
		}
	}
	return ""
}

func htmlUnescapeAttr(s string) string {
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	return s
}

func (s *Service) enrichItemMedia(items []rssItem, ogLimit int) []rssItem {
	if len(items) == 0 {
		return items
	}
	out := make([]rssItem, len(items))
	copy(out, items)
	remaining := ogLimit
	for i := range out {
		if out[i].PublisherHost == "" {
			out[i].PublisherHost = extractPublisherHost(out[i].RawXML, out[i].Link, out[i].Source)
		}
		if out[i].Thumbnail == "" {
			out[i].Thumbnail = extractThumbnailFromItemXML(out[i].RawXML)
		}
		if out[i].Thumbnail == "" && remaining > 0 && out[i].Link != "" && !isAggregatorHost(hostFromURL(out[i].Link)) {
			if img := s.fetchOGImage(out[i].Link); img != "" {
				out[i].Thumbnail = img
				remaining--
			}
		}
	}
	return out
}