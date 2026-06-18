<!-- Generated: 2026-06-18 | Files scanned: 142 | Token estimate: ~750 -->

# Dependencies

## External Services (APIs)

| Service | Purpose | Config Key | Client |
|---------|---------|------------|--------|
| **Google Gemini** | Primary LLM (gemini-3.1-flash-lite) | `GEMINI_API_KEY` + `_2..5` | `Gemini/internal/providers/gemini.go` |
| **OpenRouter** | Fallback LLM (free model rotation) | `OPENROUTER_API_KEY` + `_2..5` | `Gemini/internal/providers/openrouter.go` |
| **SerpAPI** | Live market/web search | `SERPAPI_KEY` + `_2` | `Gemini/internal/tools/handlers/search.go` |
| **Tavily** | AI-powered search | `TAVILY_KEY` + `_2` | `Gemini/internal/tools/handlers/search.go` |
| **Redis** | Session persistence | `REDIS_ADDR` (default: `127.0.0.1:6379`) | `Gemini/internal/redis/client.go` |

## Go Dependencies

### Direct Dependencies
| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/redis/go-redis/v9` | v9.20.0 | Redis client (session store) |
| `github.com/google/generative-ai-go` | v0.20.1 | Google Gemini API client |
| `github.com/ledongthuc/pdf` | latest | PDF text extraction |
| `github.com/nguyenthenguyen/docx` | latest | DOCX text extraction |
| `github.com/xuri/excelize/v2` | v2.10.1 | XLSX/Excel file parsing |
| `golang.org/x/time` | v0.15.0 | Rate limiting |

### Key Indirect Dependencies
| Package | Purpose |
|---------|---------|
| `google.golang.org/api` | Google API client libraries |
| `google.golang.org/grpc` | RPC framework |
| `go.opentelemetry.io/otel` | OpenTelemetry tracing |
| `github.com/google/uuid` | UUID generation |
| `golang.org/x/oauth2` | OAuth2 authentication |

## Frontend Dependencies

### Runtime Dependencies
| Package | Version | Purpose |
|---------|---------|---------|
| `marked` | ^14.0.0 | Markdown → HTML rendering |
| `prismjs` | ^1.29.0 | Syntax highlighting |
| `isomorphic-dompurify` | ^2.15.0 | XSS-safe HTML sanitization |
| `nanostores` | ^1.3.0 | Reactive state management |

### Development Dependencies
| Package | Version | Purpose |
|---------|---------|---------|
| `vite` | ^6.0.0 | Build tool and dev server |
| `typescript` | ^5.6.0 | Type checking |
| `@playwright/test` | ^1.60.0 | E2E testing |
| `@types/node` | ^25.9.3 | Node.js types |
| `eslint` | ^9.39.4 | Code linting |
| `prettier` | ^3.8.4 | Code formatting |

## Tool Integrations

### File Processing Tools
```go
// Document parsers
- PDF: ledongthuc/pdf
- DOCX: nguyenthenguyen/docx  
- XLSX: xuri/excelize/v2
- Plain text: Go standard library
```

### Data Processing
```
// Search tools
- Web search: SerpAPI
- AI search: Tavily
- Market data: Yahoo Finance API (built-in)
```

### Caching & Performance
```
// LRU Cache
- Tool result caching: internal/cache/lru.go
- Redis: Session persistence
- In-memory fallback: sync.Map
```

## Build & Deployment

### Build Tools
| Tool | Purpose |
|------|---------|
| Go 1.25.6 | Backend compilation |
| Vite 6.x | Frontend bundling |
| TypeScript 5.6 | Type checking |
| Playwright 1.60 | E2E testing |

### Deployment Platform
- **Render** - Cloud hosting
- Single binary serves API + static files
- Redis as external service (optional)

## Configuration Management

### Environment Variables
```env
# Required
GEMINI_API_KEY=your_key_here
OPENROUTER_API_KEY=your_key_here  
SERPAPI_KEY=your_key_here
TAVILY_KEY=your_key_here

# Optional
REDIS_ADDR=127.0.0.1:6379
USE_OPENROUTER_ONLY=0
```

### Runtime Configuration
- API key rotation support (multiple keys per service)
- Redis connection fallback to in-memory
- Feature flags via environment variables

## Security Dependencies
- `isomorphic-dompurify` - XSS protection
- CSP headers for external resources
- Rate limiting on API endpoints
- Input validation on all handlers