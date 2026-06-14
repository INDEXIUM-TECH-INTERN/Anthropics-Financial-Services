# Code Review — 2026-06-14 (Local Review)

**Reviewed**: 2026-06-14
**Branch**: main (uncommitted changes)
**Decision**: APPROVE with comments

## Summary

Các thay đổi chủ yếu là **refactoring backend Go** — thêm real streaming support, cải thiện security headers, rate limiting, path traversal protection, và gỡ bỏ frontend source files khỏi `dist/` (chỉ giữ built assets). Code chất lượng tốt, build pass, `go vet` clean.

---

## Findings

### CRITICAL
**None**

### HIGH
**None**

### MEDIUM

1. **CORS origin có thể bị override bởi biến môi trường mà không có validation**
   - File: [server.go:210-222](Gemini/internal/api/server.go#L210-L222)
   - `ALLOWED_ORIGIN` env var được set trực tiếp vào response header mà không validate format. Nếu attacker kiểm soát env var (ví dụ qua docker-compose), có thể inject arbitrary origin.
   - **Khuyến nghị**: Validate origin format (phải bắt đầu bằng `http://` hoặc `https://`), hoặc whitelist các origin hợp lệ.

2. **SSE endpoint không có authentication**
   - File: [handlers.go:39-77](Gemini/internal/api/handlers.go#L39-L77)
   - `/api/events` SSE endpoint mở hoàn toàn (`Access-Control-Allow-Origin: *`), không có rate limiting (được skip ở middleware). Có thể bị abuse để tạo nhiều kết nối SSE dài hạn (slowloris-style).
   - **Khuyến nghị**: Thêm connection limit per IP, hoặc yêu cầu session token.

3. **`handleChatStream` không trả về HTTP error khi stream fail**
   - File: [handlers.go:258-367](Gemini/internal/api/handlers.go#L258-L367)
   - Khi `ProcessMessageStream` trả về error, handler chỉ gửi error qua SSE chunk nhưng không set HTTP status code. Client HTTP thông thường (không phải EventSource) sẽ không biết request thất bại.
   - **Khuyến nghị**: Đối với non-SSE clients, cần set `w.WriteHeader(http.StatusInternalServerError)` trước khi gửi error chunk.

4. **`streamFinalResponse` trong orchestrator dùng goroutine + channel cho streaming nhưng không có backpressure**
   - File: [orchestrator.go:151-180](Gemini/internal/core/orchestrator.go#L151-L180)
   - Goroutine gọi `GenerateStream` và gửi vào `streamDone` channel. Nếu provider stream chậm, goroutine sẽ block trên `streamDone <- err` sau khi stream xong (channel unbuffered). Điều này tạo ra goroutine leak nếu caller đã timeout và không đọc từ channel nữa.
   - **Khuyến nghị**: Dùng buffered channel (`make(chan error, 1)`) — đã làm rồi ✅. Tuy nhiên, goroutine vẫn có thể leak nếu `GenerateStream` không bao giờ trả về. Cần thêm `context.WithTimeout` cho stream operation.

### LOW

1. **`fmt.Printf` thay vì structured logging**
   - Files: Tất cả Go files
   - Hiện tại dùng `fmt.Printf` cho logging. Trong production production, nên dùng structured logger (zap, slog) để dễ filter và parse.
   - **Khuyến nghị**: Chuyển sang `log/slog` (Go 1.21+) hoặc zap.

2. **Emoji trong code comments và log messages**
   - Files: Tất cả Go files (⚠️, ❌, ✅, 🔑, 📩, v.v.)
   - Emoji trong log output gây khó khăn khi parse log bằng tools (grep, awk, ELK stack).
   - **Khuyến nghị**: Dùng text-based log levels (`[INFO]`, `[ERROR]`, `[WARN]`) thay emoji.

3. **`handleChatStream` lưu history sau khi stream xong — có thể mất data nếu client disconnect**
   - File: [handlers.go:345-365](Gemini/internal/api/handlers.go#L345-L365)
   - History được save sau khi stream done. Nếu client disconnect giữa chừng, partial response vẫn được lưu. Điều này có thể gây inconsistent state.
   - **Khuyến nghị**: Chỉ save history khi stream hoàn toàn thành công (không error).

4. **`generatePPTXWithScript` dùng `exec.Command("python", ...)` mà không validate script path**
   - File: [dispatcher.go:455-499](Gemini/internal/core/dispatcher.go#L455-L499)
   - Nếu `REPORT_GENERATOR_PATH` env var bị set đến arbitrary path, có thể execute arbitrary script.
   - **Khuyến nghị**: Validate script path nằm trong project directory.

5. **Frontend dist files bị xóa — cần xác nhận đây là intentional cleanup**
   - 20 files trong `frontend/dist/src/` bị xóa (main.js, services, stores, components, utils)
   - Chỉ giữ lại `index.html` và built assets (`.js`, `.css` bundles)
   - **Cần xác nhận**: Nếu đây là migration từ Vite dev mode sang production build-only, thì OK. Nếu không có build step trong CI/CD, frontend sẽ thiếu code.

---

## Validation Results

| Check | Result |
|---|---|
| `go build ./...` | ✅ Pass |
| `go vet ./...` | ✅ Pass |
| Server health check | ✅ Pass (HTTP 200) |
| `go test` | ⚠️ Skipped (no test files found) |

---

## Files Reviewed

| File | Change Type | Notes |
|---|---|---|
| `Gemini/cmd/gemini-cli/main.go` | Modified | Clean, simple CLI entry point |
| `Gemini/internal/api/handlers.go` | Modified | Added streaming, config keys, validation helpers |
| `Gemini/internal/api/server.go` | Modified | Added security headers, rate limiting, CORS, auth middleware |
| `Gemini/internal/core/agent.go` | Modified | Clean facade, proper mutex usage |
| `Gemini/internal/core/dispatcher.go` | Modified | Map-based dispatch, path traversal protection, file size limits |
| `Gemini/internal/core/orchestrator.go` | Modified | Added real streaming with GenerateStream |
| `Gemini/internal/providers/openrouter.go` | Modified | Added GenerateStream with SSE parsing |
| `frontend/dist/index.html` | Modified | Updated asset references |
| `frontend/dist/src/*` (20 files) | **Deleted** | Source files removed from dist — intentional cleanup? |
| `frontend/dist/assets/main-*.js` | Modified | New build output |
| `frontend/dist/assets/main-*.css` | Modified | New build output |

---

## Next Steps

1. ✅ Code quality tốt, không có CRITICAL/HIGH issues
2. ⚠️ Cần xác nhận việc xóa frontend source files khỏi `dist/` là intentional
3. 💡 Cân nhắc thêm tests cho streaming handler và path validation
4. 💡 Nếu deploy production, nên thêm structured logging và connection limits cho SSE
