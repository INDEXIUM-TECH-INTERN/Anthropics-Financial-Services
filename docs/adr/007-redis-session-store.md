# ADR-007: Redis Session Store

## Status

Accepted

## Context

The AI agent maintains conversation history in memory. Without persistence:
- All context is lost on server restart.
- Multi-instance deployments (Render, Kubernetes) cannot share sessions.
- Users cannot resume conversations across browser sessions.

## Decision

Use **Redis** as the session store backend, with in-memory fallback for local development.

### Architecture

```
ProcessMessage()
  → sessionStore.GetSession(sessionID)
    → [Redis] HGET "session:{id}" "history"
    → [Fallback] In-memory map
  → agent.LoadHistory(msgs)
  → orchestrator.ProcessMessage()
  → sessionStore.SaveSession(sessionID, session)
    → [Redis] HSET "session:{id}" "history" {serialized}
    → [Fallback] In-memory map
```

### Session Data Model

| Field | Type | Description |
|-------|------|-------------|
| `ID` | string | Session identifier (UUID or user-provided) |
| `History` | []Message | Full conversation history |
| `CreatedAt` | time.Time | Session creation timestamp |
| `UpdatedAt` | time.Time | Last activity timestamp |

### Key Format

```
session:{sessionID}:history  — JSON-serialized message array
session:{sessionID}:metadata — session metadata
```

### Environment Configuration

| Variable | Default | Purpose |
|----------|---------|---------|
| `REDIS_URL` | (empty) | Redis connection URL. Empty = in-memory fallback |
| `SESSION_TTL` | 86400 | Session TTL in seconds (default: 24h) |

## Consequences

### Positive
- Sessions survive server restarts.
- Supports horizontal scaling (multiple instances share Redis).
- TTL-based auto-cleanup prevents unbounded memory growth.
- Graceful fallback to in-memory when Redis is unavailable.

### Negative
- Adds infrastructure dependency (Redis).
- Serialization/deserialization overhead per message.
- Network latency for Redis round-trips.

## Alternatives Considered

| Approach | Why Rejected |
|----------|-------------|
| PostgreSQL session table | Heavier dependency, overkill for ephemeral sessions |
| File-based persistence | Doesn't scale horizontally, I/O bottleneck |
| JWT-encoded sessions | Size limits, no server-side revocation |

## Related

- `Gemini/internal/store/` — Session store interface + Redis implementation
- `Gemini/internal/store/session_store_test.go` — 9 test cases with miniredis
- ADR-001: Multi-Provider Failover (session resilience aligns with provider resilience)
