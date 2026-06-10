# 📋 Indexium Financial AI Services — Project Overview

> **Full Name:** Indexium Financial AI Services (Anthropics-Financial-Services)
> **Version:** 2026.06
> **UI Language:** Vietnamese
> **Tech Stack:** Go 1.25.6 (backend) + Vanilla HTML/CSS/JS (frontend)

---

## 📑 Table of Contents

1. [Introduction](#1-introduction)
2. [Architecture Overview](#2-architecture-overview)
3. [Detailed Workflow](#3-detailed-workflow)
4. [API Endpoints](#4-api-endpoints)
5. [Agent System & Routing](#5-agent-system--routing)
6. [Tools](#6-tools)
7. [LLM Providers & Failover](#7-llm-providers--failover)
8. [Context Window Management](#8-context-window-management)
9. [Session Storage](#9-session-storage)
10. [Frontend](#10-frontend)
11. [Security](#11-security)
12. [Environment Configuration](#12-environment-configuration)
13. [How to Run](#13-how-to-run)
14. [Evaluator (Testing)](#14-evaluator-testing)
15. [Directory Structure](#15-directory-structure)

---

## 1. Introduction

**Indexium Financial AI Services** is a full-stack financial AI agent workspace that provides intelligent financial research, analysis, and advisory capabilities through a conversational interface. It is designed for Vietnamese financial professionals and investors.

### Key Features

| Feature | Description |
|---------|-------------|
| 🤖 **Multi-Agent Routing** | Automatically routes queries to 1 of 10 specialized financial agents |
| 🔄 **ReAct Loop** | Reasoning + Acting: LLM reasons → calls tools → processes results → reasons again |
| 🔍 **Real-time Market Data** | Live market search via Google Search (SerpAPI) & Tavily |
| 🌐 **Web Scraping** | Deep web content extraction with SSRF protection |
| 📄 **Document Loading** | Loads agent/skill markdown from GitHub repository |
| 💬 **Multi-Chat Sessions** | Multiple independent conversations with Redis persistence |
| 📡 **SSE Real-time Updates** | Live pipeline telemetry (agent selection, skill loading, tool execution) |
| 🔁 **Multi-Provider Failover** | Gemini → OpenRouter with 5+ free models and automatic key rotation |
| 🧠 **Context Summarization** | Automatically summarizes long conversation history to save tokens |
| 🇻🇳 **Vietnamese-First** | All prompts, UI, routing logic, and error messages in Vietnamese |

---

## 2. Architecture Overview

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
│  │              │ │             │ │                        │   │
│  │ • ReAct Loop │ │ • 6 tools   │ │ • Full history         │   │
│  │ • Routing    │ │ • LRU cache │ │ • Summarization        │   │
│  │ • Bootstrap  │ │ • Handoff   │ │ • BuildLLMHistory      │   │
│  └──────┬───────┘ └──────┬──────┘ └────────────────────────┘   │
│         │                │                                       │
│  ┌──────▼───────┐ ┌──────▼──────────────────────────────────┐   │
│  │   Routing    │ │  Tools Facade (tools/tools.go)          │   │
│  │   (core/)    │ │                                         │   │
│  │              │ │  ┌──────────┐ ┌─────────┐ ┌──────────┐ │   │
│  │ • AI Router  │ │  │ market/  │ │scraper/ │ │registry/ │ │   │
│  │ • Heuristic  │ │  │(SerpAPI, │ │(Web     │ │(GitHub   │ │   │
│  │ • Validation │ │  │ Tavily)  │ │ Scrape) │ │ Docs)    │ │   │
│  └──────────────┘ │  └──────────┘ └─────────┘ └──────────┘ │   │
│                    └─────────────────────────────────────────┘   │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  MultiProvider (providers/multiprovider.go)              │   │
│  │  ┌────────────┐  ┌─────────────┐  ┌─────────────┐       │   │
│  │  │  Gemini    │  │ OpenRouter  │  │ OpenRouter  │ ...   │   │
│  │  │  Provider  │  │ Provider #1 │  │ Provider #2 │       │   │
│  │  └────────────┘  └─────────────┘  └─────────────┘       │   │
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

---

## 3. Detailed Workflow

### 3.1. Main Chat Flow (ReAct Loop)

```
User sends message
        │
        ▼
┌─ API Server ───────────────────────────────────────────────┐
│  POST /api/chat                                            │
│  • Parse request (max 1MB body)                            │
│  • Load session from Redis (by chat_id)                    │
│  • agent.LoadHistory(session.Messages)                     │
│  • agent.ProcessMessage(message)  ← serialized by requestMu│
└────────────────────────┬───────────────────────────────────┘
                         │
                         ▼
┌─ Orchestrator ─────────────────────────────────────────────┐
│  ProcessMessage(userInput)                                 │
│                                                            │
│  ┌─ New conversation? ──────────────────────────────┐      │
│  │  YES:                                              │      │
│  │    ├─ Slash command (/earnings, /market)?          │      │
│  │    │   → Use predefined route plan                 │      │
│  │    ├─ Casual greeting (short, social)?             │      │
│  │    │   → Skip routing, enter ReAct loop directly   │      │
│  │    └─ Normal query:                                │      │
│  │        → bootstrapContextInternal()                │      │
│  │           ├─ selectRoutePlan()  [AI Router]        │      │
│  │           ├─ Load agent doc from GitHub (cached)   │      │
│  │           ├─ Load skill docs from GitHub (cached)  │      │
│  │           └─ Real-time data if needed (parallel)   │      │
│  │                                                    │      │
│  │  NO (continuing):                                  │      │
│  │    → Append message to history                     │      │
│  └────────────────────────────────────────────────────┘      │
│                         │                                    │
│                         ▼                                    │
│  ┌─ ReAct Loop (runConversationLoopInternal) ──────────┐    │
│  │                                                      │    │
│  │  ① Check context window                              │    │
│  │     └─ If over token limit → SummarizeOldest()       │    │
│  │                                                      │    │
│  │  ② BuildLLMHistory(keepRecent=7)                     │    │
│  │     ├─ [MemorySummary] (if exists)                   │    │
│  │     ├─ [Bootstrap: agent config + skill docs]        │    │
│  │     └─ [Last 7 messages]                             │    │
│  │                                                      │    │
│  │  ③ Send to LLM (provider.Generate)                   │    │
│  │     └─ MultiProvider: Gemini → OpenRouter fallback   │    │
│  │                                                      │    │
│  │  ④ AI response has tool calls?                       │    │
│  │     ├─ NO  → Return text response (done)             │    │
│  │     └─ YES → Dispatcher.HandleToolCalls()            │    │
│  │              ├─ financial_research (Google Search)    │    │
│  │              ├─ tavily_search (Tavily)                │    │
│  │              ├─ financial_scrape (Web scraping)       │    │
│  │              ├─ financial_calculate (stub)            │    │
│  │              ├─ handoff_request (agent handoff)       │    │
│  │              └─ load_financial_context (load docs)    │    │
│  │              → Append tool results to history         │    │
│  │              → Go back to step ① (loop)              │    │
│  │                                                      │    │
│  │  ⑤ Handoff?                                          │    │
│  │     └─ If agent.handoffPlan != nil                   │    │
│  │        → Load new agent, bootstrap, continue         │    │
│  │                                                      │    │
│  │  ⚡ Limit: max 2-3 tool calls per turn               │    │
│  └──────────────────────────────────────────────────────┘    │
└────────────────────────────────────────┬───────────────────────┘
                                         │
                                         ▼
┌─ API Server ───────────────────────────────────────────────┐
│  • agent.GetHistory() → updatedHistory                     │
│  • Save session to Redis                                   │
│  • Return {reply, history, metrics}                        │
└────────────────────────────────────────────────────────────┘
```

### 3.2. Routing Flow

```
User Query
    │
    ▼
┌─ selectRoutePlan() ──────────────────────────┐
│                                              │
│  ① AI-powered routing:                       │
│     • Send query + catalog to LLM            │
│     • LLM returns JSON (agent, skills,      │
│       temporal, reason)                      │
│                                              │
│  ② Parse JSON response:                      │
│     • Success → Continue                     │
│     │   Failure → Fallback to heuristic      │
│                                              │
│  ③ Heuristic fallback (Vietnamese keywords): │
│     • "ban lãnh đạo" → meeting-prep-agent    │
│     • "doanh thu", "quý" → earnings-reviewer │
│     • "định giá", "dcf" → model-builder      │
│     • "so sánh", "ngành" → market-researcher  │
│     • Temporal: "hôm nay" → realtime         │
│                 "ngày mai" → is_future        │
│                 "năm 2024" → historical       │
│                                              │
│  ④ sanitizeRoutePlan():                      │
│     • Validate agent is in whitelist         │
│     • Verify agent doc exists on GitHub      │
│     • Filter valid skills                    │
│     • Fallback if invalid                    │
│                                              │
│  Output: RoutePlan{Agent, Skills, Temporal}  │
└──────────────────────────────────────────────┘
```

### 3.3. Multi-Provider Failover Flow

```
LLM Request
    │
    ▼
┌─ MultiProvider ──────────────────────────────┐
│                                              │
│  skipPrimaryUntil > 0?                       │
│  ├─ YES → Skip primary, use fallback         │
│  └─ NO  → Try primary first                  │
│                                              │
│  Primary succeeds?                           │
│  ├─ YES → Reset failure counter, return      │
│  └─ NO  → Check for quota/rate-limit error   │
│           ├─ Quota error:                    │
│           │   skipPrimaryUntil = 5~12        │
│           │   (increases with failures)      │
│           └─ Round-robin fallbacks:          │
│               • OpenRouter key #1            │
│               • OpenRouter key #2            │
│               • OpenRouter key #3            │
│               • Each: exponential backoff    │
│                 500ms → 1s → 2s → 4s (max 5s)│
│                                              │
│  Fallback succeeds?                          │
│  ├─ YES → Halve skipPrimaryUntil             │
│  │         (gradual recovery)                │
│  └─ NO  → Return "all providers failed"      │
└──────────────────────────────────────────────┘
```

---

## 4. API Endpoints

| Method | Endpoint | Description | Request Body | Response |
|--------|----------|-------------|--------------|----------|
| `POST` | `/api/chat` | Send chat message | `{"message": "...", "chat_id": "..."}` | `{"reply": "...", "history": [...], "metrics": {...}}` |
| `GET` | `/api/chats` | List chat sessions | — | `{"chats": [{"id": "...", "title": "..."}]}` |
| `POST` | `/api/chats` | Create new session | `{"title": "..."}` | `{"id": "...", "title": "...", "messages": []}` |
| `DELETE` | `/api/chats?chat_id=...` | Delete session | — | `{"status": "deleted", "chat_id": "..."}` |
| `GET` | `/api/history?chat_id=...` | Get chat history | — | `{"history": [...]}` |
| `POST` | `/api/config/keys` | Update OpenRouter keys | `{"keys": ["key1", "key2"]}` | `{"status": "success"}` |
| `GET` | `/api/reset` | Reset conversation | — | `{"status": "reset"}` |
| `GET` | `/api/events` | SSE stream (real-time) | — | `data: {"type": "...", "payload": "..."}` |
| `GET` | `/` | Frontend static files | — | HTML/CSS/JS |

### Metrics returned per chat response

```json
{
  "latency_ms": 3500,
  "token_in": 250,
  "token_out": 500,
  "ram_mb": "45.23 MB",
  "cpu_load": "12 Goroutines (Active)"
}
```

---

## 5. Agent System & Routing

### 10 Financial Specialist Agents

| Agent | Purpose | Skills |
|-------|---------|--------|
| `pitch-agent` | Pitch decks, presentations | pitch-deck, datapack-builder, cim-builder, teaser, buyer-list, comps-analysis, precedent-transactions, lbo-model, merger-model |
| `meeting-prep-agent` | Meeting preparation | briefing-pack, biography-generator, company-profile, news-digest |
| `market-researcher` | Market research | sector-overview, competitive-analysis, comps-analysis, idea-generation, thesis-tracker, catalyst-calendar |
| `earnings-reviewer` | Earnings report analysis | earnings-analysis, earnings-preview, initiating-coverage, model-update, morning-note, xlsx-author |
| `model-builder` | Financial modeling | dcf-model, lbo-model, 3-statement-model, merger-model, xlsx-author, audit-xls |
| `valuation-reviewer` | Valuation review | valuation-review, gp-reporting, lp-reporting |
| `gl-reconciler` | General ledger reconciliation | break-detection, root-cause-analysis, sign-off-routing |
| `month-end-closer` | Month-end closing | accruals, roll-forwards, variance-commentary |
| `statement-auditor` | Statement auditing | lp-statement-audit, distribution-verification |
| `kyc-screener` | KYC screening | onboarding-doc-parsing, gap-flagging |

### Slash Commands

| Command | Target Agent |
|---------|--------------|
| `/earnings <ticker>` | `earnings-reviewer` |
| `/market <query>` | `market-researcher` |

---

## 6. Tools

### 6 Available Tools

| Tool | Description | Data Source |
|------|-------------|-------------|
| `financial_research` | Google Search via SerpAPI | SerpAPI (Google Search, Vietnamese locale) |
| `tavily_search` | Tavily Search (AI-powered) | Tavily API (advanced depth, direct answers) |
| `financial_scrape` | Deep web content extraction | HTTP GET + regex extraction (SSRF protected) |
| `financial_calculate` | Financial calculations | Stub (returns formatted string) |
| `handoff_request` | Delegate to another agent | Internal handoff mechanism |
| `load_financial_context` | Load agent/skill documents | GitHub raw content API (cached) |

### Caching

- **Search/Scrape cache:** LRU cache with max 200 entries, keyed by query/URL
- **Document cache:** In-memory cache for agent/skill docs from GitHub (avoids repeated HTTP calls)

---

## 7. LLM Providers & Failover

### Provider Interface

```go
type Provider interface {
    GenerateText(systemPrompt, userPrompt string) (string, error)
    Generate(ctx context.Context, req messaging.Request) (messaging.Message, error)
}
```

### Providers

| Provider | Description | Default |
|----------|-------------|---------|
| `GeminiProvider` | Google Generative Language API | `gemini-flash-latest` |
| `OpenRouterProvider` | OpenAI-compatible API | `meta-llama/llama-3.3-70b-instruct:free` |
| `MultiProvider` | Failover wrapper | Primary + fallbacks |

### Priority Order

1. **Gemini** (primary) — if `GEMINI_API_KEY` is set
2. **OpenRouter Key #1** (fallback)
3. **OpenRouter Key #2** (fallback)
4. **OpenRouter Key #3** (fallback)

If `USE_OPENROUTER_ONLY=1` or no Gemini key → OpenRouter becomes primary.

### Free Model Chain (OpenRouter)

The system automatically tries free models in order:
1. `nvidia/nemotron-3-super-120b-a12b:free`
2. `google/gemini-2.0-flash-exp:free`
3. `meta-llama/llama-3.3-70b-instruct:free`
4. `mistralai/mistral-7b-instruct:free`
5. `qwen/qwen-2.5-7b-instruct:free`

---

## 8. Context Window Management

### Principles

- **Full history is NEVER deleted** — kept intact for UI display
- **Summarization only affects** what is sent to the LLM

### Summarization Process

```
Context window exceeds token limit?
    │
    ├─ NO  → Send history normally
    └─ YES → SummarizeOldest()
              │
              ├─ Take old messages (keep first 2 + last 7)
              ├─ Send to LLM with summarization prompt
              ├─ Store summary in MemorySummary
              └─ BuildLLMHistory():
                  [MemorySummary] + [Bootstrap] + [Last 7 messages]
```

### Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `CONTEXT_KEEP_RECENT` | 7 | Number of recent messages to keep intact |
| `CONTEXT_MAX_TOKENS` | 92000 | Token threshold to trigger summarization |
| `CONTEXT_MAX_SUMMARY_INPUT` | 18000 | Max chars sent to summarization prompt |

---

## 9. Session Storage

### Redis (Primary)

- **Keys:** `chat:session:<id>` (data), `chat:sessions:list` (index)
- **Format:** JSON serialization
- **Fallback:** If Redis unavailable → in-memory store

### In-Memory Fallback

- Thread-safe with `sync.RWMutex`
- Data lost on server restart

### ChatSession Structure

```go
type ChatSession struct {
    ID        string
    Title     string
    Messages  []messaging.Message
    UpdatedAt time.Time
}
```

---

## 10. Frontend

### Technology

- **Vanilla HTML5/CSS3/JavaScript** — no framework
- **Marked.js** — Markdown rendering
- **SSE (Server-Sent Events)** — Real-time pipeline updates

### UI Features

| Feature | Description |
|---------|-------------|
| 💬 Chat Interface | Real-time conversational UI |
| 📋 Session Sidebar | Manage multiple conversations |
| 📊 Pipeline Timeline | Agent selection, skill loading, tool execution |
| ⚙️ Settings Modal | Configure API keys, select backend |
| 🌙 Dark/Light Theme | Toggle theme, persisted in localStorage |
| 🔗 Source Citations | Extract and display source URLs |
| 📈 Metrics Display | Latency, token usage, RAM |
| 🧪 Auto-test | 4 built-in banking query test cases |

### SSE Events

| Event Type | Description |
|------------|-------------|
| `agent_selected` | Agent chosen by router |
| `skill_loaded` | Skill document loaded |
| `tool_executed` | Tool being executed |
| `system` | System messages |
| `error` | Error notifications |

---

## 11. Security

### Implemented Measures

| Measure | Description |
|---------|-------------|
| 🛡️ SSRF Protection | `isBlockedURL()` blocks private/internal IPs in scraper |
| 📏 Request Body Limit | 1MB max on `/api/chat` |
| 🔒 CORS | Only allows POST, GET, OPTIONS |
| 🔑 URL Validation | Only allows http/https schemes |
| 🧱 IP Blocking | Blocks loopback, link-local, private ranges (10.x, 172.16-31.x, 192.168.x) |
| 🔄 Thread Safety | Mutex protection for shared state, provider access, cache |

### Notes

- `/api/config/keys` has no authentication — use only in trusted environments
- CORS origin is currently `*` — should be restricted in production

---

## 12. Environment Configuration

### `.env` File

```env
# LLM Providers
GEMINI_API_KEY=your_gemini_key
GEMINI_MODEL=gemini-2.5-flash

OPENROUTER_API_KEY=your_or_key_1
OPENROUTER_API_KEY_2=your_or_key_2
OPENROUTER_API_KEY_3=your_or_key_3
OPENROUTER_MODEL=meta-llama/llama-3.3-70b-instruct:free

# Search APIs
SERPAPI_KEY=your_serpapi_key
TAVILY_API_KEY=your_tavily_key

# Redis
REDIS_ADDR=127.0.0.1:6379
REDIS_PASSWORD=

# Options
USE_OPENROUTER_ONLY=0

# Context Window
CONTEXT_KEEP_RECENT=7
CONTEXT_MAX_TOKENS=92000
CONTEXT_MAX_SUMMARY_INPUT=18000

# Testing
SYSTEM_DATE_OVERRIDE=
```

### Key Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GEMINI_API_KEY` | — | Google AI Studio API key |
| `OPENROUTER_API_KEY` | — | OpenRouter API key (supports 3 keys) |
| `SERPAPI_KEY` | — | SerpAPI key for Google Search |
| `TAVILY_API_KEY` | — | Tavily Search API key |
| `USE_OPENROUTER_ONLY` | 0 | Bypass Gemini, use OpenRouter only |
| `SYSTEM_DATE_OVERRIDE` | — | Override current date (YYYY-MM-DD, for testing) |

---

## 13. How to Run

### Requirements

- Go 1.25.6+
- Redis (optional, falls back to in-memory)
- API keys: Gemini or OpenRouter (or both)

### Quick Start

```powershell
cd "C:\Users\Rabuno\Documents\AHihi\TestAIFinance"
.\run-server.ps1
```

Then open browser: **http://localhost:8080**

### Manual Build

```powershell
cd Gemini
go build -o server.exe cmd/gemini-cli/main.go
.\server.exe
```

### CLI Mode (no frontend needed)

```powershell
# Interactive mode
go run cmd/gemini-cli/main.go

# One-shot query
go run cmd/gemini-cli/main.go "Analyze VCB stock"
```

---

## 14. Evaluator (Testing)

### Structure

```
internal/evaluator/
├── test_cases.json      # 12 test cases for router accuracy
├── test_results.json    # Last run results
├── auto_tuner.go        # Automated test runner
└── refiner.go           # Prompt refinement based on failures
```

### Test Cases

12 test cases covering:
- Agent selection accuracy
- Temporal resolution (realtime/latest/historical)
- Vietnamese keyword matching

### Running Evaluator

```powershell
cd Gemini
go run cmd/gemini-cli/test_router.go "test query here"
```

---

## 15. Directory Structure

```
TestAIFinance/
├── docs/
│   ├── PROJECT_OVERVIEW.md          ← This file
│   ├── superpowers/
│   │   ├── plans/                   # Implementation plans
│   │   └── specs/                   # Technical specifications
│   └── User_Guide.md                # User guide
│
├── Gemini/
│   ├── .env                         # API keys (not committed)
│   ├── go.mod / go.sum              # Go module definition
│   ├── README.md                    # Quick start
│   │
│   ├── cmd/gemini-cli/
│   │   ├── main.go                  # Entry point (CLI + server mode)
│   │   └── test_router.go           # Router testing utility
│   │
│   └── internal/
│       ├── api/
│       │   ├── server.go            # HTTP server + REST endpoints
│       │   └── hub.go               # SSE event broadcast hub
│       │
│       ├── core/
│       │   ├── agent.go             # Agent struct + initialization
│       │   ├── orchestrator.go      # ReAct loop
│       │   ├── dispatcher.go        # Tool dispatching + LRU cache
│       │   ├── routing.go           # AI router + heuristic fallback
│       │   ├── context_window.go    # Context management + summarization
│       │   └── conversation.go      # Conversation state
│       │
│       ├── providers/
│       │   ├── provider.go          # Provider interface
│       │   ├── gemini.go            # Google Gemini adapter
│       │   ├── openrouter.go        # OpenRouter adapter
│       │   └── multiprovider.go     # Multi-provider failover
│       │
│       ├── models/
│       │   ├── common.go            # OpenRouter/OpenAI models
│       │   ├── gemini.go            # Gemini API models
│       │   └── messaging/           # Provider-agnostic message types
│       │       └── messaging.go
│       │
│       ├── tools/
│       │   ├── tools.go             # Tool facade
│       │   ├── market/              # Search tools
│       │   │   └── market.go        # SerpAPI + Tavily
│       │   ├── scraper/             # Web scraping
│       │   │   └── scraper.go       # SSRF-protected scraper
│       │   └── registry/            # Document loading
│       │       └── registry.go      # GitHub docs with cache
│       │
│       ├── prompt/
│       │   ├── system_prompt.txt         # Main agent system prompt (VI)
│       │   ├── grounded_system_prompt.txt # Time-aware prompt template
│       │   ├── router_system_prompt.txt   # Router AI prompt (VI)
│       │   ├── router_catalog.txt         # Agent/skill catalog
│       │   ├── router_user_prompt.txt     # Router user prompt template
│       │   ├── bootstrap_context_suffix.txt # Bootstrap instructions
│       │   └── context_summarizer.txt    # Summarization prompt
│       │
│       ├── store/
│       │   └── session_store.go     # Redis + in-memory session store
│       │
│       ├── redis/
│       │   └── client.go            # Redis connection
│       │
│       ├── utils/
│       │   ├── token.go             # Token estimation
│       │   └── utils.go             # Env loading, prompt helpers
│       │
│       └── evaluator/
│           ├── auto_tuner.go        # Automated test runner
│           ├── refiner.go           # Prompt refinement
│           ├── test_cases.json      # 12 test cases
│           └── test_results.json    # Last run results
│
├── frontend/
│   ├── index.html                   # Main HTML
│   ├── style.css                    # Glassmorphic UI styling
│   └── app.js                       # Frontend logic (chat, SSE, sessions)
│
├── run-server.ps1                   # Unified launcher script
├── PROJECT.md                       # Architecture overview
├── CONTEXT.md                       # Domain glossary
└── README.md                        # Root readme
```

---

## Recent Changelog

| Commit | Description |
|--------|-------------|
| `93a0c65` | Fix security vulnerabilities, data races, and code quality issues (SSRF protection, mutex fixes, ioutil migration, cache eviction, dead code removal) |
| `a2c7bba` | Expose thought signature in Gemini model and add settings trigger to frontend rail |
| `d4175ef` | Update Gemini/OpenRouter model defaults and improve backend stability |
| `0475316` | OpenRouter model fallback chain, exponential backoff, correct Gemini model, sidebar collapsed by default |
| `e5e1be6` | Multi-agent teamwork improvements — collapsible chat sidebar, DELETE session API, mutex-safe cache, heuristic router fallback |
