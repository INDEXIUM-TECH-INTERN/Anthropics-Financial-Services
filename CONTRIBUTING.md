# Contributing to Indexium Financial AI Agent

## Available Scripts

<!-- AUTO-GENERATED: scripts from Makefile + package.json — regenerate with /ecc:update-docs -->

### Backend (Go — `Gemini/Makefile`)

| Command | Description |
|---------|-------------|
| `make build` | Build the Go binary (`gemini`) |
| `make test` | Run all tests (`go test ./internal/... ./cmd/... ./api/...`) |
| `make test-race` | Run tests with race detector |
| `make test-cover` | Run tests with coverage report |
| `make test-verbose` | Run tests with verbose output |
| `make test-pkg PKG=<name>` | Run tests for a specific package (e.g. `make test-pkg PKG=core`) |
| `make server` | Start the backend server (`./gemini -server`) |
| `make clean` | Remove build artifacts (`gemini`, `coverage.out`) |
| `make fmt` | Format code (`gofmt -w .` + `goimports -w .`) |
| `make lint` | Run `go vet ./...` |

### Frontend (TypeScript — `frontend/package.json`)

| Command | Description |
|---------|-------------|
| `npm run dev` | Start Vite dev server (port 5173, proxies `/api` → `:8080`) |
| `npm run build` | Type-check + production build (output: `frontend/dist`) |
| `npm run preview` | Preview the production build locally |
| `npm run typecheck` | Run TypeScript type checking (`tsc --noEmit`) |

### Root (PowerShell)

| Command | Description |
|---------|-------------|
| `.\run-server.ps1` | Unified launcher — builds Go binary + starts server on :8080 + opens browser |
| `.\run-server.ps1 -Query "..."` | One-shot CLI query (no UI) |

<!-- /AUTO-GENERATED -->

## Development Setup

### Prerequisites
- **Go 1.25.6+** — [Download](https://go.dev/dl/)
- **Redis** (optional — falls back to in-memory if unavailable)
- **Node.js 18+** (for frontend development)
- **API Keys:** At minimum one of `GEMINI_API_KEY` or `OPENROUTER_API_KEY`

### Quick Start

```powershell
# 1. Configure API keys
cd Gemini
cp .env.example .env    # If .env.example exists, otherwise create manually
notepad .env            # Add your keys

# 2. Backend only
make build
make server             # Starts on http://localhost:8080

# 3. Frontend only (separate terminal, for dev with hot reload)
cd frontend
npm install
npm run dev             # Starts on http://localhost:5173

# 4. Full stack (simplest)
cd ..                   # Back to project root
.\run-server.ps1        # Builds + serves + opens browser
```

### Running Tests

```powershell
cd Gemini
make test               # All tests
make test-race          # With race detector
make test-cover         # With coverage
make test-pkg PKG=core  # Specific package
```

Before every commit, run:
```powershell
make fmt && make lint && make test
```

## Code Standards

- **Language**: Go 1.25+, Vietnamese for user-facing strings, English for code
- **Formatting**: `gofmt` + `goimports` (run `make fmt`)
- **Linting**: `go vet ./...` (run `make lint`)
- **Testing**: All new features require table-driven tests

## Architecture Principles

1. **Immutability**: Create new objects, never mutate existing ones (except conversation history which must append)
2. **Small interfaces**: Provider interface has 3 methods — keep it minimal
3. **Constructor injection**: Use `NewXxx()` functions with dependencies as parameters
4. **Error wrapping**: Always use `fmt.Errorf("context: %w", err)` for error propagation
5. **Thread safety**: Agent uses single `sync.RWMutex` — never nest locks across packages

## Commit Convention

```
feat: add new agent for X
fix: resolve deadlock in Y
refactor: split Z into focused modules
test: add tests for W
docs: update ADR for V
chore: update dependencies
```

## File Size Limits

- Functions: < 50 lines
- Files: < 400 lines (if larger, extract a module)
- Packages: cohesive responsibility (one concern per package)

## Testing Requirements

- Table-driven tests preferred
- Use `MockProvider` from `internal/providers/mock.go` for unit tests
- Run `make test` before every commit
