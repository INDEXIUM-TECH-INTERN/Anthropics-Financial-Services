# Data Layer

<!-- Generated: 2026-06-14 | Token estimate: ~600 -->

## Persistence Strategy
**Redis-primary / in-memory fallback** — no traditional RDBMS.

## Redis Schema

### Session Store
Key pattern: `session:{chat_id}`  
Value: JSON-serialized session data (messages[], title, timestamps)

| Operation | Redis Command | File |
|-----------|--------------|------|
| Get session | `GET session:{id}` | `internal/store/session_store.go` |
| Set session | `SET session:{id} EX 86400` | `internal/store/session_store.go` |
| Delete session | `DEL session:{id}` | `internal/store/session_store.go` |
| List sessions | `KEYS session:*` | `internal/store/session_store.go` |

### Session Data Structure
```json
{
  "chat_id": "uuid",
  "title": "Cuộc trò chuyện mới",
  "messages": [
    {
      "role": "user" | "assistant",
      "content": "string",
      "timestamp": "ISO-8601",
      "attachments": []
    }
  ],
  "created_at": "ISO-8601",
  "updated_at": "ISO-8601"
}
```

## LRU Cache (Tool Results)
- Located in: `internal/cache/lru.go`
- Caches tool call results (search, scrape, market data) to avoid redundant API calls
- Max entries: 256 (default)
- TTL: per-entry expiration

## In-Memory Fallback
When Redis is unavailable (connection refused, timeout):
- `SessionStore` automatically falls back to `map[string]*Session`
- Data lost on server restart
- Logged warning: `⚠️ [Redis] ... falling back to in-memory only`

## Pub/Sub (In-Process)
- `internal/pubsub/hub.go` — in-process pub/sub, **not** Redis pub/sub
- Used for SSE event fan-out within the single server process
- Channels: `subscribe chan<- Event`, `unsubscribe chan<- int`

## Configuration (Runtime)
- API keys stored in `os.Getenv()` at startup
- Runtime key updates via `PUT /api/config/keys` → stored in-memory (not persisted)
- Env file: `Gemini/.env` (not committed)

## Files Referencing Data Layer
| File | Purpose |
|------|---------|
| `internal/store/session_store.go` | Session CRUD interface + Redis/in-mem impl |
| `internal/store/session_store_test.go` | Session store unit tests |
| `internal/redis/client.go` | Redis connection pool |
| `internal/cache/lru.go` | LRU cache for tools |
| `internal/cache/lru_test.go` | LRU cache unit tests |
| `internal/pubsub/hub.go` | In-process event hub |
