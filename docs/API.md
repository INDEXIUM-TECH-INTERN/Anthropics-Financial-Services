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

## CORS

All endpoints include: `Access-Control-Allow-Origin: *`, Methods: `POST, GET, OPTIONS`, Headers: `Content-Type, Authorization`.
