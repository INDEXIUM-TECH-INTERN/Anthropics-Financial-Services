# Environment Configuration

> All configuration is via environment variables. Copy `Gemini/.env.example` to `Gemini/.env` and fill in your values.

<!-- AUTO-GENERATED: env vars from .env.example — regenerate with /ecc:update-docs -->

## Required Variables

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `GEMINI_API_KEY` | Yes* | Google Gemini API key (primary provider) | `AIzaSy...` |
| `OPENROUTER_API_KEY` | Yes* | OpenRouter API key (fallback provider) | `sk-or-v1-...` |

> *At least one of `GEMINI_API_KEY` hoặc `OPENROUTER_API_KEY` phải được set. Nếu chỉ có OpenRouter, set `USE_OPENROUTER_ONLY=1`.

## Provider Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `GEMINI_MODEL` | No | `gemini-3.1-flash-lite` | Gemini model name |
| `OPENROUTER_MODEL` | No | `meta-llama/llama-3.3-70b-instruct:free` | OpenRouter model name |
| `USE_OPENROUTER_ONLY` | No | `0` | Set to `1` để skip Gemini và dùng OpenRouter làm primary |

## Multi-Key Failover

Multiple API keys có thể được cấu hình cho automatic failover khi rate limits bị hit:

| Variable | Required | Description |
|----------|----------|-------------|
| `GEMINI_API_KEY_2` | No | Second Gemini key (failover) |
| `GEMINI_API_KEY_3` | No | Third Gemini key (failover) |
| `GEMINI_API_KEY_4` | No | Fourth Gemini key (failover) |
| `GEMINI_API_KEY_5` | No | Fifth Gemini key (failover) |
| `GEMINI_API_KEY_6` | No | Sixth Gemini key (failover) |
| `OPENROUTER_API_KEY_2` | No | Second OpenRouter key (failover) |
| `OPENROUTER_API_KEY_3` | No | Third OpenRouter key (failover) |
| `OPENROUTER_API_KEY_4` | No | Fourth OpenRouter key (failover) |
| `OPENROUTER_API_KEY_5` | No | Fifth OpenRouter key (failover) |

## Search Tools

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `SERPAPI_KEY` | No | SerpAPI key for Google Search (Vietnamese locale) | `abc123...` |
| `SERPAPI_KEY_2` | No | Second SerpAPI key (failover) | `def456...` |
| `TAVILY_API_KEY` | No | Tavily API key for AI-powered search | `tvly-dev-...` |
| `TAVILY_API_KEY_2` | No | Second Tavily key (failover) | `tvly-pro-...` |

## Redis Session Store

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `REDIS_ADDR` | No | `127.0.0.1:6379` | Redis server address |

When Redis is unavailable, system sẽ fall back về in-memory session storage (data mất khi restart).

## Server Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PORT` | No | `8080` | HTTP server port |
| `ALLOWED_ORIGIN` | No | `http://localhost:8080` | CORS allowed origin |

## Context Window Management

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `CONTEXT_KEEP_RECENT` | No | `7` | Recent messages kept intact during summarization |
| `CONTEXT_MAX_TOKENS` | No | `92000` | Token threshold for triggering summarization |
| `CONTEXT_MAX_SUMMARY_INPUT` | No | `18000` | Max characters sent to LLM for summarization |

<!-- /AUTO-GENERATED -->

## Example `.env` File

```env
# Primary provider (at least one required)
GEMINI_API_KEY=AIzaSy...
OPENROUTER_API_KEY=sk-or-v1-...

# Models
GEMINI_MODEL=gemini-3.1-flash-lite
OPENROUTER_MODEL=openrouter/free

# Search tools
SERPAPI_KEY=abc123...
TAVILY_API_KEY=tvly-dev-...

# Redis (optional)
REDIS_ADDR=127.0.0.1:6379

# Server
PORT=8080
ALLOWED_ORIGIN=http://localhost:8080
```

<!-- /AUTO-GENERATED -->
