# Backend — Go API Server

<!-- Generated: 2026-06-14 | Go files: 42 | Token estimate: ~950 -->

## Entry Point
`Gemini/cmd/gemini-cli/main.go` — starts CLI or HTTP server based on args.

## Route Map

| Method | Path | Handler | Auth | Description |
|--------|------|---------|------|-------------|
| GET | `/health` | `handleHealth` | — | Liveness check |
| GET | `/api/events` | `handleSSE` | — | SSE event stream (chat updates) |
| POST | `/api/chat` | `handleChat` | Rate-limit | Synchronous chat |
| POST | `/api/chat/stream` | `handleChatStream` | Rate-limit | Streaming chat (SSE) |
| GET | `/api/history` | `handleHistory` | Rate-limit | Get chat history |
| POST | `/api/reset` | `handleReset` | Rate-limit | Reset session |
| GET | `/api/chats` | `handleChats` | Rate-limit | List sessions |
| PUT | `/api/config/keys` | `handleConfigKeys` | X-Config-Secret / localhost | Update API keys at runtime |

## Middleware Chain
```
Request → rateLimitMiddleware → securityHeaders → handler
```
- **rateLimitMiddleware**: 20 req/s, burst 50; skips `/api/events`, `/health`, and non-API routes
- **securityHeaders**: CSP, X-Frame-Options, Referrer-Policy, Permissions-Policy

## Core Packages

### `internal/api/` — HTTP Layer
- `server.go` — `StartServer()`, middleware, route registration, CORS, graceful shutdown
- `handlers.go` — All HTTP handler functions

### `internal/core/` — Business Logic
- `agent.go` — `Agent` struct: `ProcessMessage()`, `ProcessMessageStream()`
- `orchestrator.go` — ReAct loop: Think → Route → Act → Observe
- `routing.go` — `Router.SelectAgent()` — AI-powered + heuristic agent selection
- `dispatcher.go` — `Dispatcher.Dispatch()` — tool execution with LRU cache
- `provider_mgr.go` — Provider selection and failover orchestration
- `context_window.go` — LLM-based context summarization for long conversations
- `bootstrap.go` — Agent initialization with system prompts
- `conversation.go` — Conversation message types
- `slash.go` — Slash command parsing (`/reset`, `/help`, etc.)

### `internal/providers/` — LLM Clients
- `provider.go` — `ProviderInterface` (Generate, RegisterKey)
- `gemini.go` — Google Gemini API client
- `openrouter.go` — OpenRouter API client (free model rotation)
- `multiprovider.go` — `MultiProvider` — failover + key rotation across both providers
- `mock.go` — Mock provider for testing

### `internal/tools/` — Tool System
- `tools.go` — Tool interface definition
- `handlers/handlers.go` — Search, Scrape, Calculate, Handoff tool implementations
- `market/market.go` — Market data helpers
- `scraper/scraper.go` — SSRF-protected web scraping
- `registry/registry.go` — Tool registration and lookup

### `internal/store/` — Persistence
- `session_store.go` — `SessionStore` interface: Get, Set, Delete, List — Redis + in-memory

### `internal/models/` — Data Types
- `common.go` — Shared request/response structs
- `gemini.go` — Gemini-specific API types
- `messaging/messaging.go` — Message, Attachment types

### `internal/cache/` — Caching
- `lru.go` — LRU cache for tool results (avoids redundant API calls)

### `internal/pubsub/` — Event System
- `hub.go` — Pub/Sub hub for SSE fan-out (subscribe, broadcast, unsubscribe)

### `internal/redis/` — Redis Client
- `client.go` — Redis connection management

### `internal/routing/` — Pre-processing
- `greeting.go` — Greeting detection (bypass agent for simple hellos)
- `temporal.go` — Temporal expression detection

### `internal/errors/` — Error Handling
- `errors.go` — Custom error types

### `internal/logger/` — Logging
- `logger.go` — Structured logging

### `internal/prompt/` — System Prompts
- Vietnamese-language system prompts for all specialist agents

### `internal/utils/` — Utilities
- `token.go` — Token estimation
- `parser.go` — Text parsing helpers
- `utils.go` — Environment loading, string utilities

### `internal/evaluator/` — Testing
- Automated evaluation suite

### `internal/scripts/` — Scripting
- `parser/parser.go` — Script parser
- `report_gen/report_gen.go` — Report generation

## Agent Routing
```
User Message → greeting check → temporal check → Router.SelectAgent()
    ├── AI-powered routing (LLM classifies intent)
    └── Heuristic fallback (keyword matching)
    
Specialist agents: market-research, earnings, modeling, valuation,
                   audit, kyc, portfolio, news, general
```

## Session Lifecycle
```
1. Client sends message with optional chat_id
2. Server loads session from Redis (or creates new)
3. Agent processes message (ReAct loop)
4. Updated session saved to Redis
5. Response + history returned to client
6. SSE events broadcast during processing
```
