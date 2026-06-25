package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"gemini-cli/internal/models/messaging"
	"gemini-cli/internal/redis"
	"golang.org/x/time/rate"
)

type AgentInterface interface {
	Reset()
	ProcessMessage(context.Context, string, []messaging.Attachment) (string, error)
	ProcessMessageStream(context.Context, string, []messaging.Attachment, func(string, bool)) error
	GetHistory() []messaging.Message
	LoadHistory([]messaging.Message)
	SetGeminiKeys([]string)
}

type ChatRequest struct {
	Message     string                 `json:"message"`
	ChatID      string                 `json:"chat_id,omitempty"`
	Attachments []messaging.Attachment `json:"attachments,omitempty"`
}

type Metrics struct {
	LatencyMs int64  `json:"latency_ms"`
	TokenIn   int    `json:"token_in"`
	TokenOut  int    `json:"token_out"`
	RamMB     string `json:"ram_mb"`
	CpuLoad   string `json:"cpu_load"`
}

type ChatResponse struct {
	Reply   string              `json:"reply"`
	History []messaging.Message `json:"history"`
	Metrics Metrics             `json:"metrics"`
	Error   string              `json:"error,omitempty"`
}

// securityHeaders adds essential security headers to all responses
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		// CSP: strict but allows CDN resources needed by the app
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' https://cdn.jsdelivr.net; "+
				"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://cdn.jsdelivr.net; "+
				"img-src 'self' data: https:; "+
				"font-src 'self' https://fonts.gstatic.com https://cdn.jsdelivr.net; "+
				"connect-src 'self' https://generativelanguage.googleapis.com; "+
				"frame-src 'none'; object-src 'none'; base-uri 'self'; form-action 'self'")
		next.ServeHTTP(w, r)
	})
}

// rateLimitMiddleware creates a per-IP rate limiter. SSE and static routes
// are excluded so long-lived streams are never 403'd by a shared budget.
func rateLimitMiddleware(next http.Handler) http.Handler {
	// Global API limiter: 20 req/s, burst 50 — applies only to /api/* (excludes SSE)
	apiLimiter := rate.NewLimiter(20, 50)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip rate limiting for SSE streams, static files, and health checks
		if strings.HasPrefix(r.URL.Path, "/api/events") || strings.HasPrefix(r.URL.Path, "/health") || !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		if !apiLimiter.Allow() {
			http.Error(w, `{"error":"Too many requests"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// getSystemMetrics returns cached system metrics, refreshing at most once per second.
// This avoids calling runtime.ReadMemStats (which triggers stopTheWorld) on every request.
var (
	cachedRAM       string
	cachedCPU       string
	cacheMetricsMu  sync.Mutex
	cacheMetricsAt  time.Time
)

func getSystemMetrics() (string, string) {
	cacheMetricsMu.Lock()
	defer cacheMetricsMu.Unlock()
	if time.Since(cacheMetricsAt) < time.Second {
		return cachedRAM, cachedCPU
	}
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	cachedRAM = fmt.Sprintf("%.2f MB", float64(m.Alloc)/1024/1024)
	cachedCPU = fmt.Sprintf("%d Goroutines (Active)", runtime.NumGoroutine())
	cacheMetricsAt = time.Now()
	return cachedRAM, cachedCPU
}

func StartServer(agent AgentInterface) {
	if err := redis.Init(); err != nil {
		fmt.Printf("⚠️ [Redis] %v — falling back to in-memory only\n", err)
	} else {
		defer redis.Close()
	}

	mux := http.NewServeMux()

	// API routes — each handler is in handlers.go
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/api/events", handleSSE)
	mux.HandleFunc("/api/reset", handleReset(agent))
	mux.HandleFunc("/api/chats", handleChats)
	mux.HandleFunc("/api/chat", handleChat(agent))
	mux.HandleFunc("/api/chat/stream", handleChatStream(agent))
	mux.HandleFunc("/api/history", handleHistory(agent))
	mux.HandleFunc("/api/config/keys", requireConfigSecret(handleConfigKeys(agent)))
	mux.HandleFunc("/api/world-news", handleWorldNews)
	mux.HandleFunc("/api/world-news/dates", handleWorldNewsDates)
	mux.HandleFunc("/api/world-news/favicon", handleWorldNewsFavicon)
	mux.HandleFunc("/api/world-news/image", handleWorldNewsImage)

	// Static files
	frontendDir := resolveFrontendDir()
	fmt.Printf("📂 [Server] Serving static files from: %s\n", frontendDir)
	mux.Handle("/", http.FileServer(http.Dir(frontendDir)))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	handler := rateLimitMiddleware(securityHeaders(mux))
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       120 * time.Second,
		IdleTimeout:       600 * time.Second,
		// Gỡ bỏ WriteTimeout vì nó làm ngắt kết nối SSE dài hạn
	}

	fmt.Printf("🚀 [Server] Backend Go is running on http://0.0.0.0:%s\n", port)
	// Graceful shutdown on SIGINT/SIGTERM
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigChan
		fmt.Printf("\n🛑 [Server] Received signal %v, shutting down gracefully...\n", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			fmt.Printf("❌ [Server] Shutdown error: %v\n", err)
		}
	}()
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("❌ [Error] Server failed: %v\n", err)
	}
	fmt.Println("✅ [Server] Stopped.")
}

func resolveFrontendDir() string {
	// Check for Vite build output first, then fall back to dev paths
	// Prioritize the original frontend outside the Gemini folder ("../frontend/dist" and "../frontend")
	for _, dir := range []string{"../frontend/dist", "../frontend", "frontend/dist", "frontend", "../../frontend"} {
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}
	return "frontend"
}

// requireConfigSecret wraps a handler with a simple shared-secret check.
// The client must provide the secret via X-Config-Secret header.
// If CONFIG_KEYS_SECRET env var is not set, only localhost requests are allowed.
func requireConfigSecret(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		secret := os.Getenv("CONFIG_KEYS_SECRET")
		if secret != "" {
			if r.Header.Get("X-Config-Secret") != secret {
				http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
				return
			}
		} else {
			// No secret configured — restrict to localhost only
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				// If RemoteAddr is malformed, reject the request
				http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
				return
			}
			if host != "127.0.0.1" && host != "::1" && host != "localhost" {
				http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
				return
			}
		}
		next(w, r)
	}
}

func enableCORS(w http.ResponseWriter, r *http.Request) {
	origin := os.Getenv("ALLOWED_ORIGIN")
	if origin == "" {
		origin = r.Header.Get("Origin")
	}
	if origin == "" {
		origin = "http://localhost:8080"
	}
	// Only set CORS for matching origin, never wildcard with credentials
	if origin != "*" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token")
	w.Header().Set("Access-Control-Max-Age", "86400")
}

// generateTitleIfNeeded creates a nice title from the first user message if still default.
func generateTitleIfNeeded(currentTitle, userMsg string) string {
	if currentTitle != "" && currentTitle != "Cuộc trò chuyện mới" {
		return currentTitle
	}
	runes := []rune(userMsg)
	if len(runes) > 60 {
		return string(runes[:57]) + "..."
	}
	if userMsg != "" {
		return userMsg
	}
	return "Cuộc trò chuyện mới"
}
