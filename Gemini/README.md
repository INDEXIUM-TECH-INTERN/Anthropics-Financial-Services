# Gemini Backend — Indexium Financial AI

Backend Go cho **Anthropics-Financial-Services**: chat AI, SSE, session store, và **Bản tin Tài chính Thế giới**.

## Chạy nhanh

```powershell
# Từ thư mục gốc repo (khuyến nghị)
cd "C:\indexium\Term 1\Anthropics-Financial-Services"
.\run-server.ps1              # :8080
.\run-server.ps1 -Port 3000   # port tùy chọn

# Chỉ backend (từ thư mục Gemini)
cd Gemini
go run ./cmd/gemini-cli --server
go run ./cmd/gemini-cli --server --port 3000
```

Mở trình duyệt: `http://localhost:<port>` → tab **Bản tin Thế giới**.

## Cấu trúc chính

| Thư mục | Vai trò |
|---------|---------|
| `cmd/gemini-cli/` | Entry point — `--server`, `--port`, one-shot query |
| `internal/api/` | HTTP server, SSE, world-news handlers |
| `internal/worldnews/` | Morning digest (CNBC, RSS, AI summary) |
| `internal/core/` | Agent, orchestrator, router |
| `internal/prompt/` | System prompts (gồm `world_news_highlight_summary.txt`) |

## Tài liệu liên quan

- [docs/WORLD_NEWS.md](../docs/WORLD_NEWS.md) — API bản tin, cache v27, nguồn dữ liệu
- [docs/ENV.md](../docs/ENV.md) — `PORT`, `ALLOWED_ORIGIN`, API keys
- [docs/RUNBOOK.md](../docs/RUNBOOK.md) — vận hành, health check, xử lý lỗi
- [docs/API.md](../docs/API.md) — REST endpoints

## Test

```powershell
cd Gemini
go test ./internal/worldnews/... -short
go test ./internal/api/... -short
```