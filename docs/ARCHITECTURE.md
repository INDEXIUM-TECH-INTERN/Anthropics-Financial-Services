# Architecture & Workflows

> Deep-dive into the system architecture, ReAct loop, routing, and failover flows.

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Frontend (HTML/CSS/JS)                       │
│  ┌──────────┐  ┌──────────┐  ┌───────────┐  ┌───────────────┐  │
│  │ Chat UI  │  │ Sidebar  │  │ Pipeline  │  │   Settings    │  │
│  │          │  │(Sessions)│  │ Timeline  │  │   Modal       │  │
│  └────┬─────┘  └────┬─────┘  └─────┬─────┘  └───────┬───────┘  │
│       │              │              │                │           │
│       └──────────────┴──────┬───────┴────────────────┘           │
│                             │ REST API + SSE                     │
└─────────────────────────────┼───────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────────┐
│                    Go Backend (port 8080)                        │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  API Server (api/server.go + api/hub.go)                 │   │
│  │  • REST endpoints (/api/chat, /api/chats, ...)           │   │
│  │  • SSE Event Hub (real-time broadcast)                   │   │
│  │  • Static file serving (frontend/)                       │   │
│  └──────────────────────┬───────────────────────────────────┘   │
│                         │ AgentInterface                         │
│  ┌──────────────────────▼───────────────────────────────────┐   │
│  │  Agent (core/agent.go)                                   │   │
│  │  • Wires all subsystems together                         │   │
│  │  • Thread-safe provider access (GetProvider)             │   │
│  │  • Request serialization (requestMu)                     │   │
│  └───────┬──────────────┬─────────────────┬─────────────────┘   │
│          │              │                 │                       │
│  ┌───────▼──────┐ ┌────▼────────┐ ┌─────▼──────────────────┐   │
│  │ Orchestrator │ │  Dispatcher │ │  Context Window        │   │
│  │ (core/)      │ │  (core/)    │ │  (core/)               │   │
│  │ • ReAct Loop │ │ • 6 tools   │ │ • Full history         │   │
│  │ • Routing    │ │ • LRU cache │ │ • Summarization        │   │
│  │ • Bootstrap  │ │ • Handoff   │ │ • BuildLLMHistory      │   │
│  └──────┬───────┘ └──────┬──────┘ └────────────────────────┘   │
│         │                │                                       │
│  ┌──────▼───────┐ ┌──────▼──────────────────────────────────┐   │
│  │   Routing    │ │  Tools Facade (tools/tools.go)          │   │
│  │   (core/)    │ │                                         │   │
│  │ • AI Router  │ │  ┌──────────┐ ┌─────────┐ ┌──────────┐ │   │
│  │ • Heuristic  │ │  │ market/  │ │scraper/ │ │registry/ │ │   │
│  │ • Validation │ │  │(SerpAPI, │ │(Web     │ │(GitHub   │ │   │
│  └──────────────┘ │  │ Tavily)  │ │ Scrape) │ │ Docs)    │ │   │
│                    └─────────────────────────────────────────┘   │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  MultiProvider (providers/multiprovider.go)              │   │
│  │  • Quota-aware failover  • Round-robin fallbacks         │   │
│  │  • Exponential backoff   • Gradual recovery              │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  Session Store (store/session_store.go)                 │   │
│  │  • Redis (primary) / in-memory (fallback)               │   │
│  │  • Multi-chat CRUD                                      │   │
│  └──────────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────────┘
```

## ReAct Loop Flow

```
User sends message
        │
        ▼
API Server → POST /api/chat (max 1MB body)
  • Load session from Redis → agent.LoadHistory()
  → agent.ProcessMessage() [serialized by requestMu]
        │
        ▼
Orchestrator → ProcessMessage(userInput)
  ├─ New conversation?
  │   ├─ Slash command (/earnings, /market) → predefined route
  │   ├─ Casual greeting → skip routing, enter loop directly
  │   └─ Normal query → bootstrapContextInternal()
  │       ├─ selectRoutePlan() [AI Router + heuristic fallback]
  │       ├─ Load agent/skill docs from GitHub (cached)
  │       └─ Real-time market data if needed (parallel Google + Tavily)
  └─ Continuing → append message to history
        │
        ▼
ReAct Loop (runConversationLoopInternal)
  ① Check context window → SummarizeOldest() if over limit
  ② BuildLLMHistory(keepRecent=7) → [Summary] + [Bootstrap] + [Last 7]
  ③ Send to LLM via MultiProvider (Gemini → OpenRouter fallback)
  ④ AI has tool calls?
      NO  → return text response (done)
      YES → Dispatcher.HandleToolCalls() → append results → loop back to ①
  ⑤ Handoff requested? → load new agent, bootstrap, continue
  ⚡ Max 2-3 tool calls per turn
        │
        ▼
API Server → agent.GetHistory() → save to Redis → return {reply, history, metrics}
```

## Routing Flow

```
User Query
    │
    ▼
selectRoutePlan()
  ① AI routing: send query + catalog to LLM → parse JSON response
  ② Parse failure → heuristic fallback (Vietnamese keyword matching)
  ③ sanitizeRoutePlan(): validate agent whitelist, verify GitHub docs, filter valid skills
    │
    ▼
Output: RoutePlan{Agent, Skills, Temporal{Intent, ResolvedDate, IsFuture}, Reason}
```

## Multi-Provider Failover

```
LLM Request
    │
    ▼
skipPrimaryUntil > 0? → Skip primary, use fallback
    │
    ▼
Try primary → Success? → Reset failures, return result
    │
    ▼
Quota/rate-limit error? → skipPrimaryUntil = 5~12 (increases with failures)
    │
    ▼
Round-robin fallbacks (OpenRouter #1, #2, #3)
  • Each: exponential backoff (500ms → 1s → 2s → 4s, max 5s) + jitter
  • Fallback succeeds → halve skipPrimaryUntil (gradual recovery)
  • All fail → return "all providers failed"
```

## Component Map

| Component | File | Responsibility |
|-----------|------|----------------|
| **API Server** | `api/server.go` | HTTP routing, SSE hub, static files, request validation |
| **Agent** | `core/agent.go` | Central orchestrator wiring all subsystems |
| **Orchestrator** | `core/orchestrator.go` | ReAct loop, context bootstrapping, handoff execution |
| **Dispatcher** | `core/dispatcher.go` | Tool execution, LRU caching, handoff handling |
| **Router** | `core/routing.go` | AI-powered agent selection + heuristic fallback |
| **Context Window** | `context_window.go` | History management, summarization, token estimation |
| **MultiProvider** | `providers/multiprovider.go` | LLM failover, quota detection, exponential backoff |
| **Session Store** | `store/session_store.go` | Redis persistence with in-memory fallback |
