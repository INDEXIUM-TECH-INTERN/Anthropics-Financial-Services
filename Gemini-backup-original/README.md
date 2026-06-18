# Gemini Financial AI Agent (Anthropics-Financial-Services)

## Tài liệu quan trọng

- **Hướng dẫn sử dụng cho người mới bắt đầu**: [docs/User_Guide.md](docs/User_Guide.md)
- **Technical Document - Context Window** (chỉ giải thích phần context window): [docs/Technical_Context_Window.md](docs/Technical_Context_Window.md)

## Chạy nhanh

```powershell
cd "C:\indexium\Term 2\Anthropics-Financial-Services\Gemini"
.\run.ps1
```

Sau đó mở trình duyệt: http://localhost:8080

## Cấu trúc thư mục chính

- `cmd/gemini-cli/` — Entry point, server, logic chính (agent, context window, orchestrator)
- `internal/core/` — ContextWindow, Agent, Orchestrator, Dispatcher
- `internal/store/` — Redis session store (lưu nhiều đoạn chat)
- `internal/prompt/` — Các prompt hệ thống
- `frontend/` — Giao diện web (được Go serve)
- `docs/` — Tài liệu (User Guide + Technical Context Window)

## Lưu ý

- Technical Document chỉ tập trung giải thích **Context Window** như yêu cầu.
- User Guide dành cho người mới, hướng dẫn chạy và sử dụng cơ bản.
