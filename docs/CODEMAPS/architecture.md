# Architecture — Indexium Financial AI

<!-- Generated: 2026-06-14 | Files scanned: 80+ | Token estimate: ~900 -->

## System Type
Full-stack monorepo — Go backend + Vanilla TypeScript frontend, single-process deployment.

## High-Level Diagram

```
┌─────────────────────────────────────────────────────────────┐
│  Browser                                                    │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  frontend/src/main.ts        (app bootstrap)         │   │
│  │  ├── components/chat/*        (UI rendering)         │   │
│  │  ├── components/sidebar/*     (conversation list)    │   │
│  │  ├── services/sse-manager.ts (SSE event stream)      │   │
│  │  ├── services/api.ts         (REST calls)            │   │
│  │  └── stores/app-state.ts     (global state)          │   │
│  └──────────────────────┬───────────────────────────────┘   │
│                         │ SSE + REST                         │
└─────────────────────────┼───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│  Go API Server (:8080)                                      │
│  Gemini/internal/api/server.go                              │
│  ├── securityHeaders  (CSP, HSTS, etc.)                     │
│  ├── rateLimitMiddleware (20 req/s, burst 50)               │
│  └── mux                                                         │
│      ├── GET  /health           → handleHealth              │
│      ├── GET  /api/events       → handleSSE (real-time)     │
│      ├── POST /api/chat         → handleChat (sync)         │
│      ├── POST /api/chat/stream  → handleChatStream          │
│      ├── GET  /api/history      → handleHistory             │
│      ├── POST /api/reset        → handleReset               │
│      ├── GET/POST /api/chats    → handleChats (CRUD)        │
│      └── PUT  /api/config/keys  → handleConfigKeys          │
│                                                             │
│  Agent (core/agent.go)                                      │
│  └── Orchestrator (core/orchestrator.go)  ← ReAct Loop     │
│      ├── Router       (core/routing.go)    → agent select   │
│      ├── Dispatcher   (core/dispatcher.go) → tool dispatch  │
│      ├── ProviderMgr  (core/provider_mgr.go)                │
│      │   ├── Gemini   (providers/gemini.go)                 │
│      │   └── OpenRouter (providers/openrouter.go)           │
│      ├── ContextWindow (core/context_window.go)             │
│      └── SessionStore (store/session_store.go)              │
│          ├── Redis (redis/client.go)                        │
│          └── In-memory fallback                             │
│                                                             │
│  Tools (internal/tools/)                                    │
│  ├── Search   (SerpAPI / Tavily)                            │
│  ├── Scrape   (SSRF-protected HTTP fetch)                   │
│  ├── Market   (market data helpers)                         │
│  ├── Calculate (expression eval)                            │
│  └── Registry (tools/registry/registry.go)                  │
│                                                             │
│  Pub/Sub Hub (pubsub/hub.go)  ← SSE fan-out                │
└─────────────────────────────────────────────────────────────┘
```

## Data Flow (Chat Request)

1. `POST /api/chat/stream` → `handleChatStream`
2. Validate auth, rate-limit check
3. Load session from Redis/memory via `SessionStore`
4. `Agent.ProcessMessageStream()` → `Orchestrator.Run()`
5. **ReAct Loop** (repeat until final answer):
   - **Think**: Call LLM (Gemini → OpenRouter failover) with system prompt + history
   - **Route**: `Router.SelectAgent()` picks specialist agent (market, earnings, KYC…)
   - **Act**: `Dispatcher.Dispatch()` → executes tool (Search, Scrape, Market, Calculate)
   - **Observe**: Tool result appended to context
6. Each token + tool event → `Hub.Broadcast()` → SSE → frontend `sse-manager.ts`
7. Frontend renders tokens in `message-bubble.ts`, tool calls in `thinking-card.ts`
8. Final response saved to `SessionStore`

## ReAct Loop Detail

```
User Message
    │
    ▼
┌──────────┐    ┌──────────┐    ┌──────────┐
│  Think   │───▶│   Act    │───▶│ Observe  │
│ (LLM)    │    │ (Tool)   │    │(result)  │
└──────────┘    └──────────┘    └──────────┘
     ▲                               │
     └────────── loop ←──────────────┘
     (until LLM returns final answer)
```

## Provider Failover Chain

```
Gemini API ──fail──▶ OpenRouter (model 1) ──fail──▶ OpenRouter (model 2) → … (5 keys)
     │                      │                         │
     └──── quota/rate ──────┴──── quota/rate ─────────┘
```

## Key Design Decisions
- **SSE for streaming** — real-time token + tool event delivery, no polling
- **Pub/Sub Hub** — decouples SSE broadcast from agent execution
- **Provider rotation** — 2 Gemini keys + 5 OpenRouter keys for rate-limit resilience
- **Redis for sessions** — survives server restart; in-memory fallback if Redis down
- **Vanilla TypeScript frontend** — no framework, minimal bundle
