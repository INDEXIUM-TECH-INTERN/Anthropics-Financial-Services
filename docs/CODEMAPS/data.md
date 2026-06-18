<!-- Generated: 2026-06-18 | Files scanned: 142 | Token estimate: ~600 -->

# Data Layer

## Persistence Strategy
**Redis-primary / in-memory fallback** - No traditional RDBMS. Automatic fallback when Redis unavailable.

## Redis Schema

### Session Store
Key pattern: `chat:session:{chat_id}`  
Value: JSON-serialized session data with 24h TTL

| Operation | Redis Command | Implementation |
|-----------|--------------|----------------|
| Get session | `GET chat:session:{id}` | `store.GetSession()` |
| Set session | `SET chat:session:{id} EX 86400` | `store.SaveSession()` |
| Delete session | `DEL chat:session:{id}` | `store.DeleteSession()` |
| List sessions | `SMEMBERS chat:sessions:list` | `store.ListSessions()` |

### Session Data Structure
```json
{
  "id": "chat_123abc",
  "title": "Cuộc trò chuyện mới",
  "messages": [
    {
      "role": "user" | "assistant" | "system" | "tool",
      "content": "string",
      "tool_calls": [...],
      "tool_responses": [...],
      "latency_ms": 150,
      "token_in": 100,
      "token_out": 200,
      "ram_mb": "128MB",
      "cpu_load": "15%"
    }
  ],
  "updated_at": "2026-06-18T10:30:00Z"
}
```

## Storage Architecture

### Primary Storage (Redis)
```
Connection Pool → Session CRUD → JSON Serialization ↔ Redis
    ↓
  Automatic failover to in-memory on connection loss
```

### Fallback Storage (In-Memory)
```
sync.Map → Session Cache → LRU Eviction (1000 sessions max)
    ↓
  Data persists only during server runtime
```

## Caching Layer

### Tool Results Cache (`internal/cache/lru.go`)
- Purpose: Cache expensive tool results (search, scrape, market data)
- Max entries: 256
- Eviction: LRU when full
- TTL: Per-tool configuration

## Message Types (`internal/models/messaging/`)
```typescript
type Message struct {
  Role: 'user' | 'assistant' | 'system' | 'tool'
  Content: string
  ToolCalls: ToolCall[]
  ToolResponses: ToolResponse[]
  LatencyMs: number
  TokenIn: number
  TokenOut: number
  RamMB: string
  CpuLoad: string
}
```

## Event System (In-Process)
```
pubsub/hub.go → Event Channels → SSE Broadcasting
Event Types: system, token, tool_call, done, error
```

## Configuration Storage
- API Keys: Environment variables (runtime only)
- Runtime Updates: `POST /api/config/keys` (in-memory, not persisted)
- Session Keys: Generated UUID with `chat_` prefix

## Key Files
- `Gemini/internal/store/session_store.go` - Session CRUD implementation
- `Gemini/internal/store/memory.go` - In-memory fallback
- `Gemini/internal/redis/client.go` - Redis connection management
- `Gemini/internal/cache/lru.go` - Tool result caching
- `Gemini/internal/pubsub/hub.go` - Event broadcasting

## Data Flow
```
User Request → Session Load → Message Processing → Session Save
    ↓              ↓                ↓              ↓
  GetSession → ProcessMessage → SaveSession → Broadcast Event
```

## Migration History
- Initial: In-memory only
- Added: Redis with 24h TTL
- Added: Automatic failover
- Added: Tool result caching
- Added: Performance metrics tracking