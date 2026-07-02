# Hướng Dẫn Đóng Góp (CONTRIBUTING)

> Documentation generated from source-of-truth files. Regenerate with `/ecc:update-docs`.

<!-- AUTO-GENERATED: scripts table and environment docs regenerated 2026-06-18 -->
<!-- /AUTO-GENERATED -->

Chào mừng bạn đến với dự án TestAIFinance! Tài liệu này hướng dẫn bạn cách đóng góp vào dự án.

## 🚀 Bắt Đầu

### Yêu cầu Cầu Thiết

- **Node.js** (>= 18.0.0) cho frontend
- **Go** (>= 1.25.6) cho backend
- **Redis** (>= 6.0) - tùy chọn, fallback về in-memory nếu không có

### Cài đặt

```bash
# Clone repository
git clone <repository-url>
cd TestAIFinance

# Cài đặt frontend dependencies
cd frontend
npm install

# Quay về thư mục gốc
cd ..

# Cài đặt Go dependencies (backend sẽ tự động download khi build)
cd Gemini
go mod tidy
```

## 📋 Bảng Lệnh Tham Chiếu

<!-- AUTO-GENERATED: scripts from package.json — regenerate with /ecc:update-docs -->

### Frontend (TypeScript/Vite)

| Command | Mô tả |
|---------|-------|
| `cd frontend && npm run dev` | Khởi động dev server với hot reload |
| `cd frontend && npm run build` | Build production với type checking |
| `cd frontend && npm run preview` | Preview production build |
| `cd frontend && npm run typecheck` | Kiểm tra type chỉ (không build) |
| `cd frontend && npm run lint` | Chạy ESLint |
| `cd frontend && npm run lint:fix` | Tự động fix ESLint |
| `cd frontend && npm run format` | Format code với Prettier |
| `cd frontend && npm run format:check` | Kiểm tra formatting Prettier |
| `cd frontend && npm run test` | Chạy test suite với Vitest |
| `cd frontend && npm run test:watch` | Chạy tests trong watch mode |
| `cd frontend && npm run test:e2e` | Chạy E2E tests với Playwright |
| `cd frontend && npm run prepare` | Setup Husky pre-commit hooks |

### Backend (Go)

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

### Full Stack (PowerShell)

```powershell
# Chạy cả backend + frontend
.\run-server.ps1
.\run-server.ps1 -Port 3000
```

<!-- /AUTO-GENERATED -->

## 🧪 Quy trình Testing

### Frontend Testing

- **Unit tests**: Sử dụng Vitest với React Testing Library
- **E2E tests**: Sử dụng Playwright
- **Coverage**: Tối thiểu 80% coverage cho các features mới

```bash
# Chạy unit tests
cd frontend && npm test

# Chạy E2E tests (đảm bảo dev server đang chạy)
cd frontend && npm run test:e2e

# Chạy với UI mode
cd frontend && npm run test:e2e -- --ui
```

### Backend Testing

- **Unit tests**: Sử dụng Go testing package
- **Integration tests**: Sử dụng test containers cho database
- **Coverage**: Sử dụng `go tool cover`

```bash
# Chạy tất cả tests
cd Gemini && make test

# Chạy với race detector (detects data races)
cd Gemini && make test-race

# Xem coverage report
cd Gemini && make test-cover
```

## 💅 Code Style

### Frontend

- **ESLint 9**: Config đã sẵn có trong `frontend/.eslintrc.js`
- **Prettier**: Config đã sẵn có trong `frontend/.prettierrc`
- **Husky**: Pre-commit hooks tự động chạy linter và formatter

```bash
# Fix auto-format
cd frontend && npm run lint:fix
cd frontend && npm run format

# Check formatting
cd frontend && npm run lint
cd frontend && npm run format:check
```

### Backend

- `gofmt`: Format code tự động
- `go vet`: Static analysis
- `goimports`: Import organization

```bash
# Format code
cd Gemini && make fmt

# Lint code
cd Gemini && make lint
```

## 📝 Quy tắc Git

### Commit Messages

Sử dụng conventional commits:

```bash
feat: Thêm tính năng mới
fix: Sửa lỗi
refactor: Tái cấu trúc code
docs: Cập nhật tài liệu
test: Thêm/sửa tests
chore: Maintenance tasks
```

### Quy trình làm việc

1. **Tạo feature branch**: `git checkout -b feature/new-feature`
2. **Commit changes**: `git commit -m "feat: add new feature"`
3. **Push**: `git push origin feature/new-feature`
4. **Create PR**: Mở PR từ GitHub

### PR Checklist

- [ ] Đã chạy tất cả tests (`npm test`, `make test`)
- [ ] Đã chạy linter (`npm run lint`, `make lint`)
- [ ] Code đã được format (`npm run format:check`, `make fmt`)
- [ ] Đã thêm/update tests cho tính năng mới
- [ ] Đã cập nhật tài liệu (nếu cần)
- [ ] PR title follows conventional commits

## 🔗 Development Workflow

### Feature Development

1. **Planning**: Tạo issue và plan trước khi bắt đầu code
2. **TDD**: Viết tests trước, implement sau
3. **Reviews**: Chờ review từ teammates
4. **Testing**: Đảm bảo tất cả tests pass
5. **Deployment**: Merge lên main sau khi review

### Bug Fixing

1. **Reproduce**: Tạo test case reproduce bug
2. **Fix**: Sửa lỗi, đảm bảo test pass
3. **Verify**: Test manual để xác nhận fix
4. **Regression**: Đảm bảo không gây lỗi mới

## 🛡️ Security

- **Never commit API keys**: Luôn check `.env` trong `.gitignore`
- **Input validation**: Validate tất cả user input
- **Use environment variables**: Không hardcode secrets
- **Dependencies audit**: Thường xuyên update dependencies

## 📚 Tài nguyên

- **Documentation**: `docs/` folder
- **Architecture**: `docs/ARCHITECTURE.md`
- **API Reference**: `docs/API.md`
- **Agent Documentation**: `docs/AGENTS.md`
- **Tools Reference**: `docs/TOOLS.md`

## 🤝 Support

Nếu bạn có câu hỏi:

1. Check existing issues trên GitHub
2. Tạo issue mới nếu chưa có
3. Tag core team để được hỗ trợ

## 🎯 Contributing Tips

1. **Start small**: PR nhỏ dễ review hơn
2. **Communicate**: Luôn update status work
3. **Tests are important**: Viết tests tốt
4. **Documentation**: Documentation là code của bạn
5. **Be patient**: Review có thể mất thời gian

---

Happy coding! 🚀