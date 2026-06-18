<!-- Generated: 2026-06-18 | Files scanned: 142 | Token estimate: ~700 -->

# Backend Architecture

## API Routes

### REST Endpoints
```
GET  /health                     → HealthCheck (server.go:27)
POST /api/chats                  → handleChats (handlers.go:93)
  ├─ GET  /api/chats             → store.ListSessions()
  ├─ POST /api/chats             → store.SaveSession(newChat)
  └─ DELETE /api/chats?chat_id=X → store.DeleteSession(chatID)

POST /api/chat                   → handleChat (handlers.go:182)
POST /api/chat/stream            → handleChatStream (handlers.go:258)
GET  /api/history?chat_id=X      → handleHistory (handlers.go:370)
POST /api/config/keys            → handleConfigKeys (handlers.go:400)
POST /api/reset                  → handleReset (handlers.go:80)
GET  /events                    → handleSSE (handlers.go:39)
```

### SSE Events
```
Client → /events → pubsub.GlobalHub → Broadcaster → All Connected Clients
Event Types: system, token, done, error
```

## Core Architecture

### Agent Flow
```
User Request → HTTP Handler → Agent Interface → Load History → ProcessMessage → Save History → Response
                                ↓
                         Streaming: ProcessMessageStream → SSE chunks → Real-time updates
```

### Key Components

#### API Layer (`Gemini/internal/api/`)
- `server.go` - HTTP server setup with CORS
- `handlers.go` - All endpoint implementations
- Agent Interface methods for HTTP integration

#### Core Layer (`Gemini/internal/core/`)
- `agent.go` - Main agent orchestration
- `orchestrator.go` - Tool coordination
- `router.go` - Message routing
- `dispatcher.go` - Task dispatching

#### Provider Layer (`Gemini/internal/providers/`)
- `gemini.go` - Gemini API client
- `provider.go` - Common provider interface
- Multi-provider support with failover

#### Storage Layer (`Gemini/internal/store/`)
- `memory.go` - In-memory session storage
- `redis.go` - Redis-backed storage
- Session CRUD operations

## Service Chain

### Chat Processing Chain
1. **HTTP Handler** (`handleChat`) - Validate request, get session
2. **Agent** (`agent.ProcessMessage`) - Load history, process with LLM
3. **Provider** (`GeminiProvider.Call`) - Execute API call
4. **Tools** (if needed) - Market data, search, calculation
5. **Storage** (`store.SaveSession`) - Persist updated history

### Streaming Chain
1. **HTTP Handler** (`handleChatStream`) - Setup SSE connection
2. **Agent** (`ProcessMessageStream`) - Stream processing
3. **SSE Hub** (`pubsub`) - Broadcast to all clients
4. **Metrics** - Real-time performance tracking

## Key Files
- `Gemini/cmd/gemini-cli/main.go` - Entry point (CLI mode or server mode)
- `Gemini/internal/api/handlers.go` - All HTTP handlers (500+ lines)
- `Gemini/internal/core/agent.go` - Main agent logic (critical path)
- `Gemini/internal/providers/gemini.go` - Gemini API integration
- `Gemini/internal/store/memory.go` - Session persistence

## Dependencies
- Go 1.25.6 (module: gemini-cli)
- go-redis/v9 (optional, for Redis backend)
- External APIs: Gemini, SerpAPI, Tavily
- SSE for real-time updates