# Runbook — Indexium Financial AI Agent

> Deployment procedures, health checks, common issues, and rollback guidance.

## Deployment

### Render.com (Production)

The project is configured for auto-deploy on [Render](https://render.com) via `render.yaml`.

**Service:** `indexium-gemini` (Go, free plan, Singapore region)

**Auto-deploy:** Triggered on every commit to `main` that changes:
- `Gemini/**`
- `frontend/**`
- `render.yaml`

**Build:**
```bash
cd Gemini && go build -o gemini ./cmd/gemini-cli
```

**Start:**
```bash
cd Gemini && ./gemini -server
```

**Health check:** `GET /health` → `200 OK` body with JSON response

**Dependencies:**
- `indexium-redis` (Key Value store, free plan, `allkeys-lru` eviction)

**Required secrets** (set in Render dashboard, `sync: false`):
- `GEMINI_API_KEY`, `GEMINI_API_KEY_2`–`KEY_6`
- `OPENROUTER_API_KEY`
- `SERPAPI_KEY`, `SERPAPI_KEY_2`
- `TAVILY_API_KEY`, `TAVILY_API_KEY_2`
- `REDIS_ADDR` (optional, defaults to 127.0.0.1:6379)
- `USE_OPENROUTER_ONLY` (optional, defaults to 0)

### Local / Self-Hosted

```powershell
# 1. Configure
cd Gemini
cp .env.example .env    # or create manually
notepad .env

# 2. Build + Run
make build
make server              # http://localhost:8080

# Or use the unified launcher
cd ..
.\run-server.ps1         # Builds + serves + opens browser
```

## Health Checks

| Check | Endpoint | Expected | Description |
|-------|----------|----------|-------------|
| Server alive | `GET /health` | `200 OK` with JSON response | Basic liveness |
| Chat working | `POST /api/chat` | `200` with `reply` field | End-to-end AI response |
| SSE connected | `GET /api/events` | `text/event-stream` | Real-time event stream |
| Sessions | `GET /api/chats` | `200` with `chats` array | Session store reachable |

### Quick Health Test

```powershell
# Server alive (should return JSON with status, version, timestamp)
curl http://localhost:8080/health | jq .

# Chat test
curl -X POST http://localhost:8080/api/chat `
  -H "Content-Type: application/json" `
  -d '{"message":"Xin chào","chat_id":"health_check"}' | jq .

# Sessions list
curl http://localhost:8080/api/chats | jq .

# For Windows without jq:
# Remove | jq from each command
```

## Monitoring

### Key Metrics (per response)

Every `POST /api/chat` response includes:

| Metric | Field | Description |
|--------|-------|-------------|
| Latency | `metrics.latency_ms` | Total request processing time |
| Input tokens | `metrics.token_in` | Estimated input token count |
| Output tokens | `metrics.token_out` | Estimated output token count |
| Memory | `metrics.ram_mb` | Current Go process memory allocation |
| Goroutines | `metrics.cpu_load` | Active goroutine count |

### Log Patterns

| Pattern | Meaning | Action |
|---------|---------|--------|
| `🚀 [Server] Backend Go is running` | Server started | — |
| `📩 [Server] Received message` | Incoming chat request | — |
| `⚠️ [Redis] ... falling back to in-memory` | Redis unavailable | Check Redis connection |
| `❌ [Server] Server failed` | Fatal server error | Check logs for cause |
| `🔑 [Config] Updated N OpenRouter keys` | Keys updated at runtime | — |
| `🛑 [Server] Received signal ... shutting down` | Graceful shutdown | — |

## Common Issues

### Server won't start

| Symptom | Cause | Fix |
|---------|-------|-----|
| `bind: address already in use` | Port 8080 occupied | Kill existing process or change `PORT` env var |
| `go: command not found` | Go not installed | Install Go 1.25.6+ from https://go.dev/dl/ |
| `.env not found` | Missing config | Copy `.env.example` to `.env` and add keys |

### AI returns errors

| Symptom | Cause | Fix |
|---------|-------|-----|
| `all providers failed` (HTTP 502) | All LLM providers exhausted | Check API keys, wait for quota reset |
| `Bad request` (HTTP 400) | Malformed JSON or missing fields | Verify request body format |
| `Message too long` (HTTP 400) | Message exceeds 50,000 chars | Shorten the message |
| Empty response | Provider returned no content | Retry, check provider status |

### Redis issues

| Symptom | Cause | Fix |
|---------|-------|-----|
| `falling back to in-memory` | Redis not running | Start Redis or ignore (sessions work in-memory) |
| Sessions lost on restart | In-memory mode | Set up Redis for persistence |

### Frontend issues

| Symptom | Cause | Fix |
|---------|-------|-----|
| Blank page | Frontend not built | Run `npm run build` in `frontend/` |
| API calls fail (CORS) | Wrong `ALLOWED_ORIGIN` | Set `ALLOWED_ORIGIN` to match your frontend URL |
| SSE not connecting | Proxy misconfigured | Ensure `/api` proxy targets `:8080` |

## Rollback

### Render.com

1. Go to Render Dashboard → `indexium-gemini` service
2. Click **"Manual Deploy"** → select previous deploy or commit hash
3. Click **"Deploy"**

### Local

```powershell
# Revert to previous commit
git log --oneline -5          # Find the target commit
git revert HEAD               # Safe rollback (creates new commit)
# OR
git reset --hard <commit>     # Hard reset (destructive)

# Rebuild
cd Gemini
make clean && make build
make server
```

## Scaling Notes

- **Current:** Single Go process, free tier on Render
- **Rate limit:** 20 req/s global, burst 50
- **Session store:** Redis (persistent) or in-memory (ephemeral)
- **Frontend:** Static files served by Go (no separate web server)
- **For higher load:** Consider adding a reverse proxy (nginx/Caddy) and running multiple Go instances behind it

## Security Checklist

- [ ] API keys set via environment variables (never hardcoded)
- [ ] `ALLOWED_ORIGIN` set to production domain (not `*`)
- [ ] Redis not exposed to public internet (`ipAllowList: []` in render.yaml)
- [ ] `.env` in `.gitignore` (never committed)
- [ ] Rate limiting active (default: 20 req/s)
- [ ] Security headers active (CSP, X-Frame-Options, etc.)
