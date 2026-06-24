# Kiến Trúc Mới cho Backend Gemini

## Tổng quan

Sau khi phân tích kiến trúc hiện tại, tôi sẽ thiết kế một kiến trúc mới dựa trên Clean Architecture principles, tối ưu cho Gemini provider và giữ nguyên functionality hiện có.

## 1. Phân tích Kiến trúc Hiện tại

### Các thành phần chính:
- **Agent**: Facade tổng điều phối, quản lý conversation và providers
- **Orchestrator**: Điều phối ReAct loop, handle tool calls
- **Dispatcher**: Xử lý tool execution với LRU cache
- **Providers**: Gemini với key rotation
- **Tools**: Financial research, scrape, calculate, handoff...
- **API Layer**: HTTP server với SSE streaming

### Vấn đề kiến trúc hiện tại:
1. **Coupling cao**: Agent phụ thuộc trực tiếp vào Orchestrator và Dispatcher
2. **Testing khó**: Dependencies hard-coded, không injectable
3. **Scalability kém**: Mở rộng mới agents/skills phải sửa nhiều chỗ
4. **Not clean**: Business logic trộn với infrastructure concerns

## 2. Kiến trúc Mới - Clean Architecture

### Layers:
```
┌─────────────────────────────────┐
│         Presentation            │
│  (HTTP API, SSE, CLI)           │
├─────────────────────────────────┤
│          Application             │
│  (Use Cases, Orchestrator)      │
├─────────────────────────────────┤
│        Domain Core              │
│  (Entities, Interfaces)        │
└─────────────────────────────────┘
           ↓
┌─────────────────────────────────┐
│        Infrastructure           │
│  (Providers, Tools, Cache, DB)   │
└─────────────────────────────────┘
```

### 2.1. Domain Layer (business logic)

#### Entities:
```go
// internal/domain/entities/
- agent.go          // Agent entity
- conversation.go   // Conversation context
- tool.go          // Tool definition
- message.go       // Message entities
```

#### Interfaces:
```go
// internal/domain/interfaces/
- agent_service.go      // Agent service interface
- provider_interface.go  // LLM provider interface
- tool_executor.go      // Tool execution interface
- repository.go         // Data access interface
```

### 2.2. Application Layer (use cases)

```go
// internal/application/
- services/
  - agent_service_impl.go     // Implement agent service
  - conversation_service.go   // Conversation management
  - orchestration_service.go  // ReAct orchestration
- use_cases/
  - chat_use_case.go       // Handle chat requests
  - tool_use_case.go       // Tool execution
  - handoff_use_case.go   // Agent handoff
- dtos/
  - request_dto.go         // Request DTOs
  - response_dto.go        // Response DTOs
```

### 2.3. Infrastructure Layer

```go
// internal/infrastructure/
- providers/
  - gemini_provider.go     // Gemini provider implementation
  - simple_gemini_pool.go  // Key pool for multiple API keys
- tools/
  - financial_tools.go     // Financial research tools
  - calculation_tools.go   // Calculation tools
  - export_tools.go        // Export tools
  - tool_registry.go       // Tool registry
- repositories/
  - conversation_repo.go   // Conversation persistence
  - cache_repository.go   // Cache implementation
- config/
  - config.go             // Configuration management
```

### 2.4. Presentation Layer

```go
// internal/presentation/
- handlers/
  - chat_handler.go       // Chat API handlers
  - stream_handler.go     // SSE streaming
  - health_handler.go     // Health check
- controllers/
  - chat_controller.go    // Chat controller
  - api_controller.go     // General API controller
- middleware/
  - auth_middleware.go    // Authentication
  - rate_limit_middleware.go // Rate limiting
  - cors_middleware.go     // CORS handling
```

## 3. Dependency Injection Container

Sử dụng interface-based dependency injection:

```go
// internal/di/
- container.go          // DI container
- bindings.go           // Service bindings
- providers.go         // Provider bindings
```

### Flow:
```
Presentation → Application → Domain ← Infrastructure
```

## 4. Agents và Skills Design

### Agents:
Mỗi agent là một service implement `AgentService` interface:

```go
type AgentService interface {
    Process(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
    GetCapabilities() []string
    Handoff(ctx context.Context, target string, payload interface{}) error
}
```

### Agents:
- **FinancialAgent**: Xử lý tài chính chung
- **ResearchAgent**: Nghiên cứu thị trường
- **EarningsAgent**: Phân tích báo cáo kết quả
- **MarketAgent**: Phân tích thị trường
- **CalculationAgent**: Tính toán tài chính

### Skills:
Skills được implement thành các tool executor:

```go
type ToolExecutor interface {
    Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error)
    GetSchema() *ToolSchema
    Validate(args map[string]interface{}) error
}
```

## 5. Gemini Provider Optimization

### Features:
- Connection pooling
- Request batching
- Rate limiting per key
- Failover handling
- Streaming optimization
- Response caching

### Configuration:
```go
type GeminiConfig struct {
    APIKeys        []string
    DefaultModel   string
    MaxRetries     int
    Timeout        time.Duration
    RateLimit      RateLimit
    Streaming      bool
    SafetySettings []SafetySetting
}
```

## 6. ReAct Loop Implementation

### ReAct Orchestrator:
```go
type ReActOrchestrator struct {
    agentService    AgentService
    toolRegistry    ToolRegistry
    maxIterations   int
    contextManager  ContextManager
}

func (r *ReActOrchestrator) Execute(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
    // Think: Call LLM
    // Act: Execute tools
    // Observe: Collect results
    // Repeat until final answer
}
```

## 7. Benefits of New Architecture

### 1. Clean Architecture:
- Business logic separated from infrastructure
- Dependencies flow inward only
- Testable components
- Framework independent

### 2. Maintainability:
- Single responsibility per component
- Clear interfaces
- Easy to modify and extend

### 3. Scalability:
- New agents can be added without modifying existing code
- Tools can be registered dynamically
- Providers can be switched easily

### 4. Testability:
- All components can be mocked
- Dependency injection enables easy testing
- Clear boundaries between layers

### 5. Performance:
- Optimized Gemini provider
- Better caching strategies
- Connection pooling
- Streaming support

## 8. Migration Plan

### Phase 1: Setup infrastructure
- Create new directory structure
- Implement DI container
- Define interfaces

### Phase 2: Domain layer
- Extract entities and interfaces
- Move business logic to domain

### Phase 3: Application layer
- Implement use cases
- Create application services

### Phase 4: Infrastructure layer
- Implement providers and tools
- Add repositories

### Phase 5: Presentation layer
- Create handlers and controllers
- Add middleware

### Phase 6: Testing
- Unit tests for all layers
- Integration tests
- Performance tests

## 9. Tools and Libraries

### Recommended:
- `go.uber.org/dig`: DI container
- `go.uber.org/zap`: Logging
- `github.com/redis/go-redis`: Redis client
- `github.com/gorilla/mux`: HTTP router
- `github.com/stretchr/testify`: Testing

### Configuration:
- Environment variables
- YAML/JSON config files
- Feature flags

## 10. Security

### Implementation:
- API key rotation
- Request validation
- Rate limiting
- Input sanitization
- CORS policies
- Security headers

### Monitoring:
- Request/response logging
- Error tracking
- Performance metrics
- Health checks