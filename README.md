# Indexium Financial AI Services

> **A Vietnamese-language Financial AI Agent** — multi-provider LLM orchestrator with ReAct tool-calling, agent routing, and real-time market data.

## 🚀 Quick Start

```powershell
# 1. Configure API keys
copy Gemini\.env.example Gemini\.env
notepad Gemini\.env   # Add your keys

# 2. Launch
.\run-server.ps1
```

Open **http://localhost:8080** in your browser.

### CLI Mode (no UI)

```powershell
cd Gemini
go run cmd/gemini-cli/main.go                        # Interactive
go run cmd/gemini-cli/main.go "Phân tích cổ phiếu VCB" # One-shot
```

---

## ✨ Features

| Feature | Description |
|---------|-------------|
| 🤖 **10 Specialist Agents** | Auto-routed: market research, earnings, modeling, valuation, audit, KYC, etc. |
| 🔄 **ReAct Loop** | Reasoning + Acting with 6 tool types per conversation turn |
| 🔍 **Live Market Data** | Google Search (SerpAPI) + Tavily Search, fetched in parallel |
| 🌐 **Web Scraping** | SSRF-protected deep content extraction |
| 💬 **Multi-Chat Sessions** | Redis-backed sessions with full CRUD |
| 📡 **Real-time Pipeline** | SSE-streamed execution timeline (agent → skills → tools) |
| 🔁 **Multi-Provider Failover** | Gemini → OpenRouter with 5+ free models, quota-aware rotation |
| 🧠 **Context Summarization** | Automatic LLM-based compression for long conversations |
| 🇻🇳 **Vietnamese-First** | All prompts, routing, UI, and error messages in Vietnamese |

---

## 🏗 Architecture

```
Frontend (Vanilla JS + SSE)
        │
        ▼
API Server (:8080) ─── SSE Hub (real-time events)
        │
        ▼
Agent ─── Orchestrator (ReAct Loop)
  │            │
  │            ├─ Router (AI-powered + heuristic fallback)
  │            ├─ Dispatcher (6 tools + LRU cache)
  │            └─ Context Window (summarization)
  │
  ├─ MultiProvider (Gemini → OpenRouter failover)
  ├─ Session Store (Redis + in-memory fallback)
  └─ Tools (Search, Scrape, Docs, Calculate, Handoff)
```

---

## 📂 Project Structure

```
TestAIFinance/
├── docs/
│   ├── ARCHITECTURE.md            # Detailed architecture & workflows
│   ├── AGENTS.md                  # Agent catalog & routing logic
│   ├── TOOLS.md                   # Tool reference
│   ├── API.md                     # API endpoints reference
│   ├── PROVIDERS.md               # LLM providers & failover
│   └── CONTEXT.md                 # Domain glossary
│
├── Gemini/
│   ├── .env                       # API keys (not committed)
│   ├── cmd/gemini-cli/
│   │   ├── main.go               # Entry point (CLI + server)
│   │   └── test_router.go        # Router testing utility
│   └── internal/
│       ├── api/                   # HTTP server + SSE hub
│       ├── core/                  # Agent, Orchestrator, Router, Dispatcher, Context
│       ├── providers/             # Gemini, OpenRouter, MultiProvider
│       ├── models/                # API request/response types
│       ├── tools/                 # Search, Scrape, Docs, Calculate
│       ├── prompt/                # System prompts (Vietnamese)
│       ├── store/                 # Redis session store
│       ├── redis/                 # Redis client
│       ├── utils/                 # Token estimation, env loading
│       └── evaluator/             # Automated test suite
│
├── frontend/
│   ├── index.html                 # Main HTML
│   ├── style.css                  # Glassmorphic UI
│   └── app.js                     # Chat, SSE, sessions, settings
│
└── run-server.ps1                 # Unified launcher
```

---

## 🛠 Prerequisites

- **Go 1.25+**
- **Redis** (optional — falls back to in-memory)
- **API Keys:**
  - [Gemini API Key](https://aistudio.google.com/app/apikey) (primary)
  - [OpenRouter API Key](https://openrouter.ai/keys) (fallback, supports 3 keys)
  - [SerpAPI Key](https://serpapi.com/) (live market search)
  - [Tavily API Key](https://tavily.com/) (AI-powered search)

---

## 📝 Configuration

All configuration is via `Gemini/.env`:

```env
GEMINI_API_KEY=your_key
OPENROUTER_API_KEY=your_key
SERPAPI_KEY=your_key
TAVILY_API_KEY=your_key
REDIS_ADDR=127.0.0.1:6379
USE_OPENROUTER_ONLY=0
```

---

## 📚 Documentation

| Document | Description |
|----------|-------------|
| [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md) | Full architecture, ReAct loop, routing flow, failover |
| [docs/AGENTS.md](./docs/AGENTS.md) | 10 specialist agents, skills, slash commands |
| [docs/TOOLS.md](./docs/TOOLS.md) | Tool reference, caching, SSRF protection |
| [docs/API.md](./docs/API.md) | REST endpoints, SSE events, metrics |
| [docs/PROVIDERS.md](./docs/PROVIDERS.md) | LLM providers, multi-provider failover, context window |
| [docs/CONTEXT.md](./docs/CONTEXT.md) | Domain glossary — key concepts and terminology |

---

## 📄 License

Internal project — Indexium Tech Internship
