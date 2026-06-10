package registry

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type LoadedDocument struct {
	DocType     string
	Name        string
	SourceURL   string
	Content     string
	ContentSize int
}

// documentCache caches loaded documents to avoid repeated GitHub HTTP calls.
// Key format: "<docType>:<name>" (e.g., "agent:market-researcher").
var (
	documentCache   = make(map[string]LoadedDocument)
	documentCacheMu sync.RWMutex
)

func LoadDocument(docType, name string) string {
	return LoadDocumentWithMetadata(docType, name).Content
}

func LoadDocumentWithMetadata(docType, name string) LoadedDocument {
	cacheKey := docType + ":" + name

	// Check cache first (fast path, no network)
	documentCacheMu.RLock()
	if cached, ok := documentCache[cacheKey]; ok {
		documentCacheMu.RUnlock()
		return cached
	}
	documentCacheMu.RUnlock()

	// Slow path: fetch from GitHub
	doc := fetchDocument(docType, name)

	// Store in cache
	documentCacheMu.Lock()
	documentCache[cacheKey] = doc
	documentCacheMu.Unlock()

	return doc
}

func fetchDocument(docType, name string) LoadedDocument {
	repoRawRoot := "https://raw.githubusercontent.com/anthropics/financial-services/main/plugins"
	docType = strings.ToLower(strings.TrimSpace(docType))
	name = strings.Trim(strings.TrimSpace(name), "/")

	docURL, err := resolveDocumentURL(repoRawRoot, docType, name)
	if err != nil {
		return LoadedDocument{
			DocType:   docType,
			Name:      name,
			SourceURL: docURL,
			Content:   err.Error(),
		}
	}

	fmt.Printf("📚 [Tool] Đang nạp %s từ GitHub: %s...\n", docType, name)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(docURL)
	if err != nil {
		return LoadedDocument{
			DocType:   docType,
			Name:      name,
			SourceURL: docURL,
			Content:   fmt.Sprintf("Lỗi kết nối khi nạp tài liệu: %v", err),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return LoadedDocument{
			DocType:   docType,
			Name:      name,
			SourceURL: docURL,
			Content:   fmt.Sprintf("Lỗi: Không tìm thấy tài liệu %s mang tên %s trên GitHub (Status: %d).", docType, name, resp.StatusCode),
		}
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return LoadedDocument{
			DocType:   docType,
			Name:      name,
			SourceURL: docURL,
			Content:   fmt.Sprintf("Lỗi khi đọc nội dung tài liệu: %v", err),
		}
	}

	return LoadedDocument{
		DocType:     docType,
		Name:        name,
		SourceURL:   docURL,
		Content:     string(content),
		ContentSize: len(content),
	}
}

func GetRoutingGuide() string {
	const readmeURL = "https://raw.githubusercontent.com/anthropics/financial-services/main/README.md"

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(readmeURL)
	if err != nil {
		return fmt.Sprintf("Lỗi kết nối khi nạp routing guide: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("Lỗi: Không tải được routing guide từ GitHub (Status: %d).", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("Lỗi khi đọc routing guide: %v", err)
	}
	return string(content)
}

func resolveDocumentURL(repoRawRoot, docType, name string) (string, error) {
	switch docType {
	case "agent":
		if name == "" {
			return "", fmt.Errorf("Lỗi: Tên agent đang trống.")
		}
		slugParts := strings.Split(name, "/")
		slug := slugParts[len(slugParts)-1]
		return fmt.Sprintf("%s/agent-plugins/%s/agents/%s.md", repoRawRoot, slug, slug), nil
	case "skill", "skills":
		parts := strings.Split(name, "/")
		if len(parts) != 2 {
			return "", fmt.Errorf("Lỗi: Tên skill phải có dạng <agent>/<skill>, ví dụ earnings-reviewer/earnings-analysis.")
		}
		return fmt.Sprintf("%s/agent-plugins/%s/skills/%s/SKILL.md", repoRawRoot, parts[0], parts[1]), nil
	default:
		return "", fmt.Errorf("Lỗi: Loại tài liệu không hợp lệ: %s", docType)
	}
}
