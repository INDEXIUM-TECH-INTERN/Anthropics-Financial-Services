package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// WithCors adds CORS headers to handlers
func WithCors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Call next handler
		next(w, r)
	}
}

// WithLogging adds logging to handlers
func WithLogging(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Log request
		fmt.Printf("%s %s %s", r.Method, r.URL.Path, r.RemoteAddr)

		// Call next handler
		next(w, r)
	}
}

// WithAuth adds authentication middleware
func WithAuth(next http.HandlerFunc, allowedKeys []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if auth is required
		if len(allowedKeys) == 0 {
			next(w, r)
			return
		}

		// Get API key from header
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			apiKey = r.URL.Query().Get("api_key")
		}

		// Validate API key
		valid := false
		for _, key := range allowedKeys {
			if key == apiKey {
				valid = true
				break
			}
		}

		if !valid {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Call next handler
		next(w, r)
	}
}

// WithRateLimit adds rate limiting middleware
func WithRateLimit(next http.HandlerFunc, requestsPerSecond int) http.HandlerFunc {
	// Simple in-memory rate limiter
	requests := make(map[string][]int64)

	return func(w http.ResponseWriter, r *http.Request) {
		// Get client IP
		ip := getClientIP(r)

		// Get current time
		now := time.Now().Unix()

		// Clean old requests
		cleanOldRequests(requests, ip, now)

		// Check if rate limit exceeded
		clientRequests := requests[ip]
		if len(clientRequests) >= requestsPerSecond {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		// Record request
		requests[ip] = append(clientRequests, now)

		// Call next handler
		next(w, r)
	}
}

// WithCompression adds compression middleware
func WithCompression(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if client accepts gzip
		acceptEncoding := r.Header.Get("Accept-Encoding")
		if strings.Contains(acceptEncoding, "gzip") {
			// Enable gzip compression
			w.Header().Set("Content-Encoding", "gzip")
		}

		// Call next handler
		next(w, r)
	}
}

// getClientIP gets the client IP address
func getClientIP(r *http.Request) string {
	// Check for forwarded IP
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Get the first IP from the list
		ip := strings.Split(forwarded, ",")[0]
		return strings.TrimSpace(ip)
	}

	// Check for real IP
	xRealIP := r.Header.Get("X-Real-IP")
	if xRealIP != "" {
		return xRealIP
	}

	// Return remote address
	return r.RemoteAddr
}

// cleanOldRequests cleans old requests from rate limiter
func cleanOldRequests(requests map[string][]int64, ip string, now int64) {
	// Remove requests older than 1 minute
	oneMinuteAgo := now - 60

	var validRequests []int64
	for _, reqTime := range requests[ip] {
		if reqTime >= oneMinuteAgo {
			validRequests = append(validRequests, reqTime)
		}
	}

	requests[ip] = validRequests
}

// logger is a simple logger
