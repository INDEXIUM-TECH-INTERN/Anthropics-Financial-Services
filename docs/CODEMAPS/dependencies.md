# Dependencies

<!-- Generated: 2026-06-14 | Token estimate: ~700 -->

## External Services (APIs)

| Service | Purpose | Config Key | Client |
|---------|---------|------------|--------|
| **Google Gemini** | Primary LLM (gemini-3.1-flash-lite) | `GEMINI_API_KEY` + `_2` | `internal/providers/gemini.go` |
| **OpenRouter** | Fallback LLM (5 free model keys) | `OPENROUTER_API_KEY` + `_2..5` | `internal/providers/openrouter.go` |
| **SerpAPI** | Live market/web search | `SERPAPI_KEY` + `_2` | `internal/tools/handlers/` |
| **Tavily** | AI-powered search | `TAVILY_API_KEY` + `_2` | `internal/tools/handlers/` |
| **Redis** | Session persistence | `REDIS_ADDR` (default: `127.0.0.1:6379`) | `internal/redis/client.go` |

## Go Dependencies

### Direct
| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/redis/go-redis/v9` | v9.20.0 | Redis client (session store) |
| `github.com/ledongthuc/pdf` | latest | PDF text extraction |
| `github.com/nguyenthenguyen/docx` | latest | DOCX text extraction |
| `github.com/xuri/excelize/v2` | v2.10.1 | XLSX/Excel file parsing |
| `golang.org/x/time` | v0.14.0 | Rate limiter (`rate.Limiter`) |

### Indirect
| Package | Purpose |
|---------|---------|
| `github.com/cespare/xxhash/v2` | Redis hashing |
| `github.com/richardlehane/mscfb` | MS compound binary format (DOC) |
| `github.com/richardlehane/msoleps` | MS OLE property sets |
| `github.com/tiendc/go-deepcopy` | Deep copy utilities |
| `go.uber.org/atomic` | Atomic operations |
| `golang.org/x/crypto` | Cryptographic utilities |
| `golang.org/x/net` | Networking |
| `golang.org/x/text` | Text processing |

## Frontend Dependencies

### Direct
| Package | Version | Purpose |
|---------|---------|---------|
| `marked` | ^14.0.0 | Markdown → HTML rendering |
| `prismjs` | ^1.29.0 | Syntax highlighting (code blocks) |
| `isomorphic-dompurify` | ^2.15.0 | XSS-safe HTML sanitization |

### Dev
| Package | Version | Purpose |
|---------|---------|---------|
| `vite` | ^6.0.0 | Dev server + production bundler |
| `typescript` | ^5.6.0 | Type checking + compilation |
| `@playwright/test` | ^1.60.0 | E2E tests |
| `@types/node` | ^25.9.3 | Node.js type definitions |
| `@types/prismjs` | ^1.26.0 | PrismJS type definitions |

## CDN Resources (loaded in frontend via CSP)
| Resource | Domain | Purpose |
|----------|--------|---------|
| PrismJS CDN | `cdn.jsdelivr.net` | Syntax highlighting CSS/JS |
| Google Fonts API | `fonts.googleapis.com` | Font loading |
| Google Fonts GStatic | `fonts.gstatic.com` | Font files |

## Build Tools
| Tool | Purpose |
|------|---------|
| Go 1.25.6 | Backend compilation (`go build`) |
| Vite 6.x | Frontend bundling |
| TypeScript 5.6 | Frontend type checking |
| Playwright 1.60 | E2E testing |
| `gofmt` / `goimports` | Go formatting |

## Deployment
- **Render** (`render.yaml`) — cloud hosting
- Single process: Go binary serves both API and static frontend files
- `run-server.ps1` — local development launcher
