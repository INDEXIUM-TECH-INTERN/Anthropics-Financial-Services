# Cập Nhật Tài Liệu - 2026-07-02

## Tóm tắt

Đồng bộ tài liệu với **Morning Digest v27** và cải tiến launcher local.

```
Created:  docs/WORLD_NEWS.md (API, cache, nguồn dữ liệu bản tin thế giới)
Updated:  README.md, CONTRIBUTING.md, docs/RUNBOOK.md, docs/ENV.md
Updated:  docs/API.md (world-news endpoints), docs/ONBOARDING.md
Updated:  Gemini/README.md, CLAUDE.md, docs/CONTRIBUTING.md
Fixed:    run-server.ps1 — $redisPort tách khỏi -Port (PowerShell case-insensitive)
Added:    --port flag (gemini-cli), run-server.ps1 -Port
```

## Thay đổi sản phẩm (tham chiếu tài liệu)

| Version | Nội dung |
|---------|----------|
| v22–v25 | Tóm tắt 800–1000 từ; CNBC stocks; tin trước 07:00; `breakingNews.date` |
| v26 | Hiển thị ngày Breaking News (API + URL fallback) |
| v27 | Tóm tắt **nhiều đoạn văn** (`\n\n`), UI render từng `<p>` |

## Chạy local — port tùy chọn

```powershell
.\run-server.ps1 -Port 3000
go run ./cmd/gemini-cli --server --port 3000
$env:PORT = "3000"; $env:ALLOWED_ORIGIN = "http://localhost:3000"
```

---

# Cập Nhật Tài Liệu - 2026-06-18

## Tóm Tắt Cập Nhật

Đã hoàn thành việc đồng bộ hóa tài liệu với codebase, tạo từ các nguồn sự thật:

```
Documentation Update
──────────────────────────────
Updated:  docs/CONTRIBUTING.md (scripts table, development environment setup)
Updated:  docs/RUNBOOK.md (deployment procedures, health checks)
Created:  Gemini/.env.example (environment variables documentation)
Flagged:  docs/AGENTS.md, docs/ARCHITECTURE.md (đã cũ nhưng cần review manual)
Skipped:  docs/API.md, docs/ENV.md (đã cập nhật gần đây)
──────────────────────────────
```

## Chi Tiết Cập Nhật

### 📝 Bảng Lệnh Script

**Frontend (TypeScript/Vite):**
| Command | Mô tả |
|---------|-------|
| `cd frontend && npm run dev` | Khởi động dev server với hot reload |
| `cd frontend && npm run build` | Build production với type checking |
| `cd frontend && npm run typecheck` | Kiểm tra type chỉ (không build) |
| `cd frontend && npm run lint` | Chạy ESLint |
| `cd frontend && npm run lint:fix` | Tự động fix ESLint |
| `cd frontend && npm run format` | Format code với Prettier |
| `cd frontend && npm run test` | Chạy test suite với Vitest |
| `cd frontend && npm run test:e2e` | Chạy E2E tests với Playwright |

**Backend (Go):**
| Command | Mô tả |
|---------|-------|
| `cd Gemini && make build` | Build binary `gemini` |
| `cd Gemini && make test` | Chạy tất cả tests |
| `cd Gemini && make test-race` | Chạy tests với race detector |
| `cd Gemini && make test-cover` | Chạy tests với coverage report |
| `cd Gemini && make test-verbose` | Chạy tests với output chi tiết |
| `cd Gemini && make test-pkg PKG=core` | Chạy tests cho package cụ thể |
| `cd Gemini && make server` | Build và chạy server |
| `cd Gemini && make clean` | Xóa build artifacts |
| `cd Gemini && make lint` | Chạy `go vet` |
| `cd Gemini && make fmt` | Format code với goimports/gofmt |

### 🌍 Môi Trường (.env.example)

Đã tạo file `Gemini/.env.example` với đầy đủ biến môi trường:

**Required:**
- `GEMINI_API_KEY` - API keys cho Google Gemini
- `SERPAPI_KEY` - Keys cho web scraping
- `TAVILY_API_KEY` - Keys cho web search

**Optional:**
- `GEMINI_MODEL` - Model configuration
- `OPENROUTER_API_KEY` - Alternative provider
- `REDIS_ADDR` - Redis connection string
- `USE_OPENROUTER_ONLY` - Provider selection

### 🛠️ Contributing Guide

Đã cập nhật `docs/CONTRIBUTING.md` với:
- Hướng dẫn setup môi trường phát triển
- Bảng lệnh script chi tiết
- Quy trình TDD và testing
- Code style enforcement (ESLint, Prettier, Husky)
- PR submission checklist
- Security guidelines

### 🚀 Runbook

Đã cập nhật `docs/RUNBOOK.md` với:
- Deployment procedures (Render.com & local)
- Health check endpoints và monitoring
- Common issues và fixes
- Rollback procedures
- Scaling notes

### ⚠️ File Cần Review Manual

Sau khi kiểm tra, các file sau đã cũ (từ 2026-06-10) nhưng cần review manual trước khi cập nhật:

- `docs/AGENTS.md` (8 ngày tuổi) - Đang được phát triển tích cực
- `docs/ARCHITECTURE.md` (8 ngày tuổi) - Kiến trúc thay đổi thường xuyên
- `docs/TOOLS.md` (8 ngày tuổi) - Tools đang được thêm mới
- `docs/PROVIDERS.md` (8 ngày tuổi) - Provider configuration thay đổi
- `docs/CONTEXT.md` (8 ngày tuổi) - Domain glossary cần cập nhật

### ✅ File Đã Cập Nhật Gần Đây

Các file sau đã được cập nhật trong 2026-06-13 và không cần thay đổi:
- `docs/API.md`
- `docs/ENV.md`
- `docs/RUNBOOK.md` (đã cập nhật lại)

### 🔧 Rules Được Tuân Thủ

- ✅ **Single source of truth**: Tất cả đều tạo từ code, không thủ công
- ✅ **Preserve manual sections**: Chỉ cập nhật phần generated, giữ lại prose viết tay
- ✅ **Mark generated content**: Sử dụng `<!-- AUTO-GENERATED -->` cho các phần tự động
- ✅ **Don't create docs unprompted**: Chỉ tạo khi có yêu cầu cụ thể

---

*Cập nhật bởi: CI/CD Documentation Update System*
*Ngày: 2026-06-18*