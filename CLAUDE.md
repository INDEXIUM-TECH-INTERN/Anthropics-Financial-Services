# CLAUDE.md — TestAIFinance

> **Indexium Financial AI Services** — Single-provider LLM orchestrator với ReAct tool-calling, key rotation, và real-time market data. Tiếng Việt first.

---

Sử dụng tiếng việt để thảo luận với Boss

## 🏗 Architecture

```
frontend/                    # Vanilla TypeScript SPA (Vite)
  src/
    app/                     # App entry, config
    entities/                # Domain entities (chat, session)
      chat/api/              # Chat history API + tests
      chat/model/            # Store, types
      session/api/           # Session CRUD + tests
      session/model/         # Store, types
    features/                # Feature modules
      chat/send/             # Send message logic + UI
      settings/modal/        # Settings modal
      sidebar/toggle/        # Sidebar toggle
      theme/toggle/          # Dark/light theme
    pages/                   # Page-level composition
    shared/                  # Shared utilities
      api/                   # HTTP client, SSE, types
      lib/                   # DOM, errors, HTML, markdown
      testing/               # Test helpers, setup
      ui/                    # Reusable UI components
    styles/                  # Global CSS

Gemini/                      # Go backend
  cmd/gemini-cli/
    main.go                  # Entry point (CLI + server)
  internal/
    api/                     # HTTP server + SSE hub
    core/                    # Agent, Orchestrator, Dispatcher, ProviderManager
    providers/               # Gemini provider với key rotation pool
    models/                  # API types
    tools/                   # Search, Scrape, Docs, Calculate
    prompt/                  # System prompts (Vietnamese)
    store/                   # Session store
    redis/                   # Redis client
    utils/                   # Token estimation, env loading, key pool
    errors/                  # Error types
    logger/                  # Logging
    evaluator/               # Automated test suite
    cache/                   # Caching layer
    routing/                 # Routing logic
    pubsub/                  # Pub/sub for SSE
    scripts/                 # Internal scripts
```

---

## 🛠 Tech Stack

| Layer           | Tech                                                     |
| --------------- | -------------------------------------------------------- |
| **Frontend**    | Vanilla TypeScript, Vite 6, Nanostores, Prism.js, Marked |
| **Backend**     | Go 1.25.6, Redis (go-redis/v9)                           |
| **Testing**     | Vitest (unit), Playwright (E2E)                          |
| **Lint/Format** | ESLint 9, Prettier, Husky + lint-staged                  |
| **Deploy**      | Render (Go service + Redis)                              |

---

## 📝 Quy ước

### Git Commit

Dùng tiền tố conventional:

- `feat:` — feature mới
- `fix:` — sửa bug
- `refactor:` — tái cấu trúc
- `docs:` — documentation
- `chore:` — maintenance
- `test:` — thêm/sửa tests

### Code Style

- **Frontend:** ESLint + Prettier (đã cấu hình sẵn)
- **Backend:** `gofmt` + `go vet`
- **TypeScript:** strict mode, `noUnusedLocals`, `noUnusedParameters`

### Đường dẫn

- Dùng đường dẫn tương đối từ project root
- Frontend aliases: `@/*` → `frontend/src/*`

---

## 🔧 Commands

### Frontend

```bash
cd frontend
npm run dev          # Dev server (Vite)
npm run build        # Production build (tsc + vite build)
npm run typecheck    # Type check only (tsc --noEmit)
npm run lint         # ESLint
npm run lint:fix     # ESLint auto-fix
npm run format       # Prettier
npm run test         # Vitest (unit)
npm run test:e2e     # Playwright (E2E)
```

### Backend

```bash
cd Gemini
go build ./...       # Build
go test ./...        # Test
go vet ./...         # Vet
go run cmd/gemini-cli/main.go              # Interactive CLI
go run cmd/gemini-cli/main.go "câu hỏi"   # One-shot
```

### Full Stack (PowerShell)

```powershell
.\run-server.ps1     # Launch cả backend + frontend
```

---

## 🔑 Environment

File: `Gemini/.env`

```env
GEMINI_API_KEY=...
GEMINI_API_KEY_2=...
GEMINI_API_KEY_3=...
GEMINI_API_KEY_4=...
GEMINI_API_KEY_5=...
SERPAPI_KEY=...
TAVILY_API_KEY=...
REDIS_ADDR=127.0.0.1:6379
GEMINI_MODEL=models/gemini-3.1-flash-lite
CONTEXT_MAX_TOKENS=92000
CONTEXT_KEEP_RECENT=7
REACT_MAX_ITERATIONS=20
STREAM_TIMEOUT_SECONDS=600
```

---

## 📚 Docs

| File                   | Mô tả                                        |
| ---------------------- | -------------------------------------------- |
| `docs/ARCHITECTURE.md` | Kiến trúc chi tiết, ReAct loop, routing flow |
| `docs/AGENTS.md`       | 10 specialist agents, skills, slash commands |
| `docs/TOOLS.md`        | Tool reference, caching, SSRF protection     |
| `docs/API.md`          | REST endpoints, SSE events, metrics          |
| `docs/PROVIDERS.md`    | Gemini provider với key rotation            |
| `docs/CONTEXT.md`      | Domain glossary                              |

---

## ⚠️ Lưu ý quan trọng

- **Không commit** `Gemini/.env` — chứa API keys
- **Không commit** `frontend/node_modules/`, `frontend/dist/`
- Frontend dùng **Vanilla TS** — không có React/Vue/Angular
- Backend dùng **Go modules** — module name: `gemini-cli`
- Redis là optional — fallback về in-memory nếu không có
- Tất cả prompts và UI messages đều **tiếng Việt**
