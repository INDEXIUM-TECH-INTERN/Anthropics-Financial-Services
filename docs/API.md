# API Reference

> REST endpoints, SSE events, request/response formats, and metrics.

## REST Endpoints

### `POST /api/chat`

Send a chat message. The main endpoint for AI interaction.

**Request:**
```json
{
  "message": "Analyse VCB stock",
  "chat_id": "chat_1234567890"
}
```

**Response:**
```json
{
  "reply": "Based on the analysis...",
  "history": [
    {"role": "user", "content": "Analyse VCB stock"},
    {"role": "assistant", "content": "Based on the analysis..."}
  ],
  "metrics": {
    "latency_ms": 3500,
    "token_in": 250,
    "token_out": 500,
    "ram_mb": "45.23 MB",
    "cpu_load": "12 Goroutines (Active)"
  }
}
```

Notes: Request body limited to 1MB. If `chat_id` is empty, defaults to `"default"`. Session history loaded from Redis before processing, saved back after.

---

### `GET /api/chats`

List all chat sessions (lightweight, no messages).

**Response:** `{"chats": [{"id": "...", "title": "...", "updated_at": "..."}]}`

---

### `POST /api/chats`

Create a new chat session. **Request:** `{"title": "New Chat"}`

---

### `DELETE /api/chats?chat_id=...`

Delete a chat session. **Response:** `{"status": "deleted", "chat_id": "..."}`

---

### `GET /api/history?chat_id=...`

Get full message history. **Response:** `{"history": [...]}`

---

### `POST /api/config/keys`

Update OpenRouter API keys at runtime. **Request:** `{"keys": ["sk-or-v1-xxx"]}`

---

### `GET /api/reset`

Reset the current in-memory conversation. **Response:** `{"status": "reset"}`

---

## SSE Events (`GET /api/events`)

Server-Sent Events stream for real-time pipeline telemetry.

- **Content-Type:** `text/event-stream`
- **Heartbeat:** `: ping` every 15 seconds

### Event Types

| Type | Description |
|------|-------------|
| `system` | Connection status |
| `agent_selected` | Agent chosen by router (`{agent, reason, skills}`) |
| `skill_loaded` | Skill document loaded (`{skill}`) |
| `tool_executed` | Tool being executed (`{tool, args}`) |
| `process` / `routing` | Processing and routing log messages |
| `success` / `error` | Status notifications |

### Example Stream

```
data: {"type":"system","payload":"SSE Connected"}
data: {"type":"routing","payload":"Analyzing request to select optimal Agent..."}
data: {"type":"agent_selected","payload":{"agent":"earnings-reviewer","reason":"Keywords: earnings","skills":["earnings-analysis"]}}
data: {"type":"tool_executed","payload":{"tool":"financial_research","args":{"query":"VCB Q2 2025"}}}
```

## Metrics

| Field | Description |
|-------|-------------|
| `latency_ms` | Total request processing time |
| `token_in` | Estimated input tokens (`len(message) / 4`) |
| `token_out` | Estimated output tokens (`len(reply) / 4`) |
| `ram_mb` | Current memory allocation |
| `cpu_load` | Active goroutine count |

## Error Codes

All error responses follow a consistent envelope:

```json
{
  "error": "error_code",
  "message": "Human-readable description"
}
```

| HTTP Status | Error Code | Meaning |
|-------------|------------|---------|
| 400 | `bad_request` | Malformed JSON or missing required fields |
| 404 | `not_found` | Chat session not found (invalid `chat_id`) |
| 429 | `rate_limited` | Too many requests (see Rate Limiting) |
| 500 | `internal_error` | Unexpected server error |
| 502 | `provider_failure` | All LLM providers failed (quota/rate-limit) |
| 503 | `overloaded` | Server at capacity, retry after backoff |

### Domain Errors

These map to the error types in `Gemini/internal/errors/`:

| Type | HTTP Status | When |
|------|-------------|------|
| `ErrProviderFailure` | 502 | All providers exhausted or quota exhausted |
| `ErrRoutingFailure` | 400 | Router cannot determine agent for query |
| `ErrContextOverflow` | 200 (internal) | Context window exceeded; summarization triggered automatically |
| `ErrSessionNotFound` | 404 | Session ID not found in Redis/memory store |
| `ErrToolExecution` | 200 (internal) | Tool execution failed; error returned to AI for retry |

## Rate Limiting

Rate limiting is applied per-IP on the `/api/chat` endpoint:

| Tier | Limit | Window |
|------|-------|--------|
| Default | 30 requests | 60 seconds |
| With valid API key | 120 requests | 60 seconds |

When the limit is exceeded:
- HTTP 429 is returned.
- Response includes `Retry-After` header (seconds until next allowed request).
- SSE connections are also subject to the same limits.

Configuration via environment variables:

| Variable | Default | Purpose |
|----------|---------|---------|
| `RATE_LIMIT` | 30 | Requests per window (per IP) |
| `RATE_LIMIT_WINDOW` | 60 | Window in seconds |
| `DISABLE_RATE_LIMIT` | (empty) | Set to `1` to disable (dev only) |

## CORS

All endpoints include: `Access-Control-Allow-Origin: *`, Methods: `POST, GET, OPTIONS`, Headers: `Content-Type, Authorization`.
