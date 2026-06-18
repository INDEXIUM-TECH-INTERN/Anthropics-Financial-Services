package config

import (
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config represents the application configuration
type Config struct {
	Server      ServerConfig      `mapstructure:"server"`
	Redis       RedisConfig       `mapstructure:"redis"`
	LLM         LLMConfig         `mapstructure:"llm"`
	Agents      AgentsConfig      `mapstructure:"agents"`
	Tools       ToolsConfig       `mapstructure:"tools"`
	Cache       CacheConfig       `mapstructure:"cache"`
	Security    SecurityConfig    `mapstructure:"security"`
	Logging     LoggingConfig     `mapstructure:"logging"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Port            string        `mapstructure:"port"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout"`
	MaxHeaderBytes  int           `mapstructure:"max_header_bytes"`
}

// RedisConfig represents Redis configuration
type RedisConfig struct {
	Addr         string `mapstructure:"addr"`
	Password     string `mapstructure:"password"`
	DB           int    `mapstructure:"db"`
	PoolSize     int    `mapstructure:"pool_size"`
	MinIdleConns int    `mapstructure:"min_idle_conns"`
}

// LLMConfig represents LLM configuration
type LLMConfig struct {
	Provider       string            `mapstructure:"provider"`
	Model          string            `mapstructure:"model"`
	Temperature    float64           `mapstructure:"temperature"`
	MaxTokens      int               `mapstructure:"max_tokens"`
	MaxRetries     int               `mapstructure:"max_retries"`
	RequestTimeout time.Duration     `mapstructure:"request_timeout"`
	StreamTimeout  time.Duration     `mapstructure:"stream_timeout"`
	RateLimit      RateLimitConfig   `mapstructure:"rate_limit"`
}

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	RequestsPerSecond int           `mapstructure:"requests_per_second"`
	BurstSize        int           `mapstructure:"burst_size"`
	Window           time.Duration `mapstructure:"window"`
}

// AgentsConfig represents agents configuration
type AgentsConfig struct {
	DefaultAgent      string   `mapstructure:"default_agent"`
	MaxIterations     int      `mapstructure:"max_iterations"`
	ContextWindow     int      `mapstructure:"context_window"`
	MaxTokens         int      `mapstructure:"max_tokens"`
	SummaryThreshold  int      `mapstructure:"summary_threshold"`
	KeepRecentMessages int     `mapstructure:"keep_recent_messages"`
}

// ToolsConfig represents tools configuration
type ToolsConfig struct {
	MaxCacheSize     int               `mapstructure:"max_cache_size"`
	CacheTTL         time.Duration     `mapstructure:"cache_ttl"`
	MaxToolExecTime  time.Duration     `mapstructure:"max_tool_exec_time"`
	AllowedTools     []string          `mapstructure:"allowed_tools"`
	DisabledTools    []string          `mapstructure:"disabled_tools"`
}

// CacheConfig represents cache configuration
type CacheConfig struct {
	Enabled      bool          `mapstructure:"enabled"`
	Type         string        `mapstructure:"type"` // "memory", "redis"
	TTL          time.Duration `mapstructure:"ttl"`
	MaxSize      int           `mapstructure:"max_size"`
	CleanupInterval time.Duration `mapstructure:"cleanup_interval"`
}

// SecurityConfig represents security configuration
type SecurityConfig struct {
	EnableCORS         bool     `mapstructure:"enable_cors"`
	AllowedOrigins     []string `mapstructure:"allowed_origins"`
	MaxRequestSize     int64    `mapstructure:"max_request_size"`
	RequireAuth       bool     `mapstructure:"require_auth"`
	APIKeyHeader      string   `mapstructure:"api_key_header"`
	AllowedAPIKeys     []string `mapstructure:"allowed_api_keys"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"` // "json", "text"
	Output     string `mapstructure:"output"` // "stdout", "file", "syslog"
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:            getEnv("PORT", "8080"),
			ReadTimeout:     getEnvDuration("READ_TIMEOUT", 30*time.Second),
			WriteTimeout:    getEnvDuration("WRITE_TIMEOUT", 120*time.Second),
			IdleTimeout:     getEnvDuration("IDLE_TIMEOUT", 600*time.Second),
			MaxHeaderBytes:  getEnvInt("MAX_HEADER_BYTES", 1<<20),
		},
		Redis: RedisConfig{
			Addr:         getEnv("REDIS_ADDR", "localhost:6379"),
			Password:     getEnv("REDIS_PASSWORD", ""),
			DB:           getEnvInt("REDIS_DB", 0),
			PoolSize:     getEnvInt("REDIS_POOL_SIZE", 10),
			MinIdleConns: getEnvInt("REDIS_MIN_IDLE_CONNS", 5),
		},
		LLM: LLMConfig{
			Provider:       getEnv("LLM_PROVIDER", "gemini"),
			Model:          getEnv("LLM_MODEL", "gemini-3.1-flash-lite"),
			Temperature:    getEnvFloat("LLM_TEMPERATURE", 0.7),
			MaxTokens:      getEnvInt("LLM_MAX_TOKENS", 8192),
			MaxRetries:     getEnvInt("LLM_MAX_RETRIES", 3),
			RequestTimeout: getEnvDuration("LLM_REQUEST_TIMEOUT", 300*time.Second),
			StreamTimeout:  getEnvDuration("LLM_STREAM_TIMEOUT", 600*time.Second),
			RateLimit: RateLimitConfig{
				RequestsPerSecond: getEnvInt("LLM_RATE_LIMIT_RPS", 10),
				BurstSize:        getEnvInt("LLM_RATE_LIMIT_BURST", 20),
				Window:          time.Minute,
			},
		},
		Agents: AgentsConfig{
			DefaultAgent:         getEnv("DEFAULT_AGENT", "financial-agent"),
			MaxIterations:        getEnvInt("MAX_ITERATIONS", 20),
			ContextWindow:        getEnvInt("CONTEXT_WINDOW", 92000),
			MaxTokens:           getEnvInt("MAX_TOKENS", 8192),
			SummaryThreshold:    getEnvInt("SUMMARY_THRESHOLD", 18000),
			KeepRecentMessages:  getEnvInt("KEEP_RECENT_MESSAGES", 7),
		},
		Tools: ToolsConfig{
			MaxCacheSize:        getEnvInt("TOOL_CACHE_MAX_SIZE", 200),
			CacheTTL:           getEnvDuration("TOOL_CACHE_TTL", 1*time.Hour),
			MaxToolExecTime:    getEnvDuration("TOOL_MAX_EXEC_TIME", 30*time.Second),
			AllowedTools:        getEnvSlice("ALLOWED_TOOLS", []string{}),
			DisabledTools:       getEnvSlice("DISABLED_TOOLS", []string{}),
		},
		Cache: CacheConfig{
			Enabled:       getEnvBool("CACHE_ENABLED", true),
			Type:          getEnv("CACHE_TYPE", "memory"),
			TTL:           getEnvDuration("CACHE_TTL", 1*time.Hour),
			MaxSize:       getEnvInt("CACHE_MAX_SIZE", 1000),
			CleanupInterval: getEnvDuration("CACHE_CLEANUP_INTERVAL", 5*time.Minute),
		},
		Security: SecurityConfig{
			EnableCORS:    getEnvBool("ENABLE_CORS", true),
			AllowedOrigins: getEnvSlice("ALLOWED_ORIGINS", []string{"*"}),
			MaxRequestSize: int64(getEnvInt("MAX_REQUEST_SIZE", 10*1024*1024)),
			RequireAuth:   getEnvBool("REQUIRE_AUTH", false),
			APIKeyHeader:  getEnv("API_KEY_HEADER", "X-API-Key"),
			AllowedAPIKeys: getEnvSlice("ALLOWED_API_KEYS", []string{}),
		},
		Logging: LoggingConfig{
			Level:      getEnv("LOG_LEVEL", "info"),
			Format:     getEnv("LOG_FORMAT", "json"),
			Output:     getEnv("LOG_OUTPUT", "stdout"),
			Filename:   getEnv("LOG_FILE", ""),
			MaxSize:    getEnvInt("LOG_MAX_SIZE", 100),
			MaxBackups: getEnvInt("LOG_MAX_BACKUPS", 3),
			MaxAge:     getEnvInt("LOG_MAX_AGE", 7),
		},
	}
}

// RedisOptions returns Redis options from config
func (c *Config) RedisOptions() *redis.Options {
	return &redis.Options{
		Addr:         c.Redis.Addr,
		Password:     c.Redis.Password,
		DB:           c.Redis.DB,
		PoolSize:     c.Redis.PoolSize,
		MinIdleConns: c.Redis.MinIdleConns,
	}
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if durationVal, err := time.ParseDuration(value); err == nil {
			return durationVal
		}
	}
	return defaultValue
}

func getEnvSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return []string{value}
	}
	return defaultValue
}