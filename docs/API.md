# API Reference

> REST endpoints, SSE events, request/response formats, and metrics.

## REST Endpoints

<!-- AUTO-GENERATED: endpoints from source — regenerate with /ecc:update-docs -->

### `POST /api/chat`

Send a chat message. The main endpoint for AI interaction (non-streaming).

**Request:**
```json
{
  "message": "Phân tích cổ phiếu VCB",
  "chat_id": "chat_1234567890",
  "attachments": []
}
```

**Response:**
```json
{
  "reply": "Based on the analysis...",
  "history": [
    {"role": "user", "content": "Phân tích cổ phiếu VCB"},
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

**Constraints:**
- Request body: max 10MB
- Message length: max 50,000 characters
- Attachments: max 10 per request
- If `chat_id` is empty, defaults to `"default"`
- Session history loaded from Redis/memory before processing, saved back after

---

### `POST /api/chat/stream`

Streaming chat endpoint — returns Server-Sent Events (SSE) with real-time token chunks.

**Request:** Same as `POST /api/chat`.

**SSE Response Stream:**
```
data: {"type":"token","text":"Based"}
data: {"type":"token","text":" on"}
data: {"type":"token","text":" the analysis..."}
data: {"type":"done","text":"Based on the analysis...","metrics":{"latency_ms":3500,"token_in":250,"token_out":500,"ram_mb":"45.23 MB","cpu_load":"12 Goroutines (Active)"}}
data: {"type":"error","error":"..."}   // only on failure
```

**Session save:** Conversation history is auto-saved to Redis/memory after stream completes.

---

### `GET /api/chats`

List all chat sessions (lightweight, no messages).

**Response:** `{"chats": [{"id": "...", "title": "...", "updated_at": "..."}]}`

---

### `POST /api/chats`

Create a new chat session.

**Request:**
```json
{"title": "New Chat"}
```

If title is empty, defaults to `"Cuộc trò chuyện mới"`.

---

### `DELETE /api/chats?chat_id=...`

Delete a chat session.

**Response:** `{"status": "deleted", "chat_id": "..."}`

---

### `GET /api/history?chat_id=...`

Get full message history for a session.

**Response:** `{"history": [...]}`

If `chat_id` is empty, returns current in-memory history.

---

### `POST /api/config/keys`

Update OpenRouter API keys at runtime (no restart required).

**Request:**
```json
{"keys": ["sk-or-v1-xxx", "sk-or-v1-yyy"]}
```

Empty keys are filtered out automatically.

---

### `GET /api/reset`

Reset the current in-memory conversation.

**Response:** `{"status": "reset"}`

---

### `GET /health`

Health check endpoint. Returns `200 OK` with body `"ok"`.

**Response:** `ok`

---

### `GET /api/world-news?date=YYYY-MM-DD`

Morning digest dashboard data (tab **Bản tin Thế giới**). Optional `date` query (ISO `YYYY-MM-DD`, GMT+7 calendar day); defaults to today.

**Response:** `WorldNewsReport` JSON — see [docs/WORLD_NEWS.md](./WORLD_NEWS.md) for full schema.

Key fields:
- `reportVersion` — cache schema version (bust cache when incremented)
- `highlightSummary` — 800–1000 words, paragraphs separated by `\n\n`
- `breakingNews[]` — `{ date, time, source, content, url, ... }` (news before 07:00 GMT+7)
- `digestWindow`, `digestUntil` — digest time range labels

---

### `GET /api/world-news/dates`

Available report dates for the date picker (90 days).

**Response:** `{ "dates": [{ "value", "label", "isToday" }], "defaultDate": "YYYY-MM-DD" }`

---

### `GET /api/world-news/favicon?host=<hostname>`

Proxy publisher favicon (avoids browser CORS).

---

### `GET /api/world-news/image?url=<encoded-url>`

Proxy article thumbnail image.

<!-- /AUTO-GENERATED -->

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

## Security Headers

All responses include the following security headers:

| Header | Value |
|--------|-------|
| `X-Content-Type-Options` | `nosniff` |
| `X-Frame-Options` | `DENY` |
| `Referrer-Policy` | `strict-origin-when-cross-origin` |
| `Permissions-Policy` | `camera=(), microphone=(), geolocation=()` |

### Content-Security-Policy

```
default-src 'self';
script-src 'self' https://cdn.jsdelivr.net;
style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://cdn.jsdelivr.net;
img-src 'self' data: https:;
font-src 'self' https://fonts.gstatic.com https://cdn.jsdelivr.net;
connect-src 'self' https://generativelanguage.googleapis.com https://openrouter.ai;
frame-src 'none'; object-src 'none'; base-uri 'self'; form-action 'self'
```

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

Rate limiting is applied **globally** (not per-IP) via `golang.org/x/time/rate`:

| Parameter | Value | Description |
|-----------|-------|-------------|
| Rate | 20 req/s | Sustained request rate |
| Burst | 50 | Maximum burst size |

When the limit is exceeded, HTTP 429 is returned with body `{"error":"Too many requests"}`.

> **Note:** The rate limiter is a simple global limiter shared across all IPs. For per-IP limiting, a middleware update is needed.

## CORS

CORS is configured via the `ALLOWED_ORIGIN` environment variable:

| Variable | Default | Description |
|----------|---------|-------------|
| `ALLOWED_ORIGIN` | `http://localhost:8080` | Allowed origin for CORS |

When set, the server sends:
- `Access-Control-Allow-Origin: <value>` (never wildcard `*`)
- `Access-Control-Allow-Credentials: true`
- Methods: `GET, POST, PUT, DELETE, OPTIONS`
- Headers: `Content-Type, Authorization, X-CSRF-Token`
- Max-Age: `86400`

SSE endpoint (`/api/events`) always allows `*` origin for compatibility.
