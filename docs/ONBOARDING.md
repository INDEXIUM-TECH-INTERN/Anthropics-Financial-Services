# Developer Onboarding Guide

> Welcome to the Indexium Financial AI Agent (TestAIFinance)!

## Quick Start

### Prerequisites
- Go 1.25.6+
- Redis (local or Docker)
- API keys: GEMINI_API_KEY or OPENROUTER_API_KEY (at minimum)

### Setup

```bash
cd Gemini
cp .env.example .env  # Add your API keys
go build -o gemini ./cmd/gemini-cli
./gemini -server  # Starts on port 8080
```

### Project Architecture

```
Gemini/
├── cmd/gemini-cli/main.go     # Entry point (CLI + server modes)
└── internal/
    ├── api/                    # HTTP server + SSE hub
    ├── core/                   # Agent, Orchestrator, Router, Dispatcher
    │   ├── slash.go           # Slash command handling (40+ commands)
    │   └── bootstrap.go       # Context bootstrapping
    ├── cache/lru.go           # Thread-safe LRU cache
    ├── errors/                # Domain error types
    ├── logger/                # Structured logging (slog)
    ├── models/                # Data models + provider-agnostic messaging
    ├── providers/             # LLM provider implementations + mock
    ├── pubsub/                # SSE event broadcasting
    ├── redis/                 # Redis client with fallback
    ├── scripts/               # Document parsing + report generation
    ├── store/                 # Session storage (Redis + in-memory fallback)
    ├── tools/                  # Tool implementations (search, scrape, calculate)
    └── utils/                 # Utilities (env loading, token estimation)
```

### Running Tests

```bash
cd Gemini
make test           # Run all tests
make test-race      # Run with race detector (requires CGO/mingw)
make test-cover     # Run with coverage report
```

### Common Workflows

1. **Adding a New Tool**: Create handler in `internal/tools/handlers/`, register in `dispatcher.go`
2. **Adding a Slash Command**: Add entry to `slashCommands` map in `internal/core/slash.go`
3. **Adding a New Agent**: Update `allowedAgents` in `routing.go` + add docs to GitHub repo
4. **Updating Prompts**: Edit files in `internal/prompt/*.txt` (use `{{PLACEHOLDER}}` syntax)

### Architecture Decisions

Read the ADRs in `docs/adr/` for rationale behind key decisions:
- ADR-001: Multi-Provider Failover
- ADR-002: ReAct Loop Architecture
- ADR-003: Context Summarization
- ADR-004: Routing Strategy
