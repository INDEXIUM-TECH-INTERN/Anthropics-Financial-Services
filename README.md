# Anthropics Financial Services

Financial AI Agent — multi-provider LLM orchestrator (Gemini, Claude, OpenRouter) with a modern web UI for Vietnamese financial research and analysis.

## 📂 Project Structure

```
├── Claude/          # Anthropic Claude direct-call example (Go)
├── Gemini/          # Full-featured agent: orchestrator, tools, multi-provider (Go)
│   ├── cmd/         # Entry points (CLI + server)
│   ├── internal/    # Core logic: providers, tools, API, evaluator
│   └── run.ps1      # Helper script to launch
├── frontend/        # Web UI (HTML/CSS/JS) — served by Gemini server
├── CONTEXT.md       # Domain glossary & architecture concepts
├── HOW_TO_RUN.md    # Step-by-step setup guide
└── README_Docs.md   # Documentation index
```

## 🚀 Features

- **Multi-provider LLM support** — Gemini (primary), OpenRouter (fallback + free models), Claude
- **Financial research tools** — live market search (SerpAPI), web scraping, calculation
- **Web UI** — chat interface with SSE live execution plan updates, Vietnamese-localized
- **CLI mode** — interactive loop or one-shot questions
- **Built-in evaluator** — automated test suite for agent quality
- **Pure Go stdlib** — no external Go dependencies, direct HTTP calls

## 🛠 Prerequisites

- **Go 1.25+**
- **API Keys:**
  - [Gemini API Key](https://aistudio.google.com/app/apikey) (required)
  - [OpenRouter API Key](https://openrouter.ai/keys) (recommended for fallbacks)
  - [SerpAPI Key](https://serpapi.com/) (optional, for live market search)

## ▶️ Quick Start

```bash
cd Gemini

# Copy and edit API keys
cp .env.example .env
# Edit .env with your keys

# Run with Web UI
go run cmd/gemini-cli/main.go -server
# Or use the helper:
./run.ps1
```

Open **http://localhost:8080** in your browser.

### CLI Mode (no UI)

```bash
cd Gemini

# Interactive loop
go run cmd/gemini-cli/main.go

# One-shot question
go run cmd/gemini-cli/main.go "So sánh tổng tài sản HDB và ACB trong 3 năm gần đây"
```

## 🏗 Architecture

| Concept | Description |
|---------|-------------|
| **Message** | Provider-agnostic conversation container (role, content, tool calls) |
| **Provider** | LLM vendor adapter — translates Messages & ToolSchemas to API calls |
| **ContextWindow** | Central conversation history — source of truth for dialogue state |
| **ToolSchema** | Neutral function definition the agent can invoke |
| **Orchestrator** | Main loop — input → memory → provider → tool dispatch |

## 📝 Notes

- Server auto-serves `../frontend` when launched from `Gemini/`
- SSE streams live execution plan updates to the right sidebar
- All prompts, routing, and tools live under `Gemini/internal/`
- Vietnamese UI — localized for Vietnamese financial research

## 📄 License

Internal project — Indexium Tech Internship
