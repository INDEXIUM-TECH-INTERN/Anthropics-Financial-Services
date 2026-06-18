# Gemini-OpenRouter

Dự án riêng biệt cung cấp dịch vụ AI thông qua OpenRouter API. Đây là một hệ thống độc lập tách biệt khỏi backend Gemini chính.

## 🚀 Cấu trúc dự án

```
Gemini-OpenRouter/
├── cmd/openrouter-cli/      # CLI entry point
├── internal/               # Internal packages
│   ├── providers/         # LLM providers (OpenRouter only)
│   ├── models/           # Data models
│   └── api/              # HTTP server (sắp tới)
├── go.mod                 # Go module
└── .env.example          # Environment template
```

## 🛠️ Công nghệ

- **Ngôn ngữ:** Go 1.25.6
- **Provider:** OpenRouter API
- **Multi-provider:** Failover tự động giữa các API keys
- **Models:** Hỗ trợ nhiều models free-tier

## 🔧 Setup

1. Copy environment file:
```bash
cp .env.example .env
```

2. Fill in your OpenRouter API keys:
```bash
OPENROUTER_API_KEY=sk-or-v1-your-key-here
OPENROUTER_API_KEY_2=sk-or-v1-secondary-key
```

## 📡 Usage

### CLI Mode
```bash
# Run with default model
go run cmd/openrouter-cli/main.go --mode cli "Hello world"

# Specify model
go run cmd/openrouter-cli/main.go --mode cli --model "gpt-4o" "Your message"
```

### Server Mode (sắp tới)
```bash
go run cmd/openrouter-cli/main.go --mode server --port 8080
```

## 🔑 Features

- **Multi-key Support:** Tự động failover khi gặp lỗi
- **Model Fallback:** Thử các models alternative khi primary fail
- **Rate Limit Handling:** Tự động xử lý rate limits
- **CLI & Server Modes:** Hỗ trợ cả CLI và HTTP server

## 📦 Dependencies

Chỉ sử dụng standard library Go. Không có external dependencies.

## ⚠️ Notes

- Project này tách biệt hoàn toàn từ Gemini backend
- Chỉ sử dụng OpenRouter API
- Dùng cho các trường hợp cần đa provider nhưng không dùng Gemini