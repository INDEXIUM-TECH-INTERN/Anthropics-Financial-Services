# Migration Plan - Codebase Refactor

**Date**: June 16, 2026
**Source**: Legacy Backend (57 files)
**Target**: New Clean Architecture (50+ files)
**Strategy**: Incremental Migration with Rollback Strategy

---

## 📊 Current State Analysis

### Legacy Codebase Structure:
```
Gemini/
├── cmd/gemini-cli/
│   └── main.go              # Entry point
├── internal/
│   ├── api/                 # HTTP server & SSE (main.go, handlers.go, server.go)
│   ├── core/                # Agent, Orchestrator, Dispatcher
│   │   ├── agent.go
│   │   ├── agent_test.go
│   │   ├── orchestrator.go
│   │   ├── dispatcher.go
│   │   └── ...
│   ├── providers/           # Gemini, OpenRouter
│   ├── tools/               # Tools system
│   ├── models/              # Data models
│   ├── models/messaging/    # Message types
│   ├── prompt/              # System prompts
│   ├── cache/               # Caching
│   ├── redis/               # Redis client
│   ├── pubsub/              # SSE pub/sub
│   ├── routing/             # Agent routing
│   ├── evaluator/           # Test suite
│   ├── scripts/             # Scripts
│   ├── utils/               # Utilities
│   ├── logger/              # Logging
│   ├── errors/              # Error handling
│   └── ...
```

### New Architecture Structure:
```
Gemini/
├── cmd/gemini-cli/
│   └── main.go              # Entry point (NEW)
├── internal/
│   ├── domain/              # Domain layer (NEW)
│   │   ├── entities/        # Entities & business logic
│   │   ├── interfaces/      # Interfaces
│   │   ├── services/        # Domain services
│   │   └── errors.go
│   ├── application/         # Application layer (NEW)
│   │   ├── services/        # Use case services
│   │   ├── use_cases/       # Use cases
│   │   └── dtos/            # DTOs
│   ├── infrastructure/      # Infrastructure layer (NEW)
│   │   ├── providers/       # LLM providers
│   │   ├── tools/           # Tool executors
│   │   ├── repositories/    # Data repositories
│   │   └── config/          # Configuration
│   ├── presentation/        # Presentation layer (NEW)
│   │   ├── handlers/        # HTTP handlers
│   │   ├── middleware/      # Middleware
│   │   └── controllers/     # API controllers
│   ├── di/                  # DI container (NEW)
│   ├── config/              # Configuration (NEW)
│   ├── cache/               # Caching (shared)
│   ├── redis/               # Redis (shared)
│   ├── pubsub/              # SSE pub/sub (shared)
│   ├── models/              # Data models (shared)
│   ├── models/messaging/    # Message types (shared)
│   ├── prompt/              # System prompts (shared)
│   ├── utils/               # Utilities (shared)
│   ├── logger/              # Logging (shared)
│   └── errors/              # Error handling (shared)
```

---

## 🎯 Migration Strategy

### **Phase 1: Foundation (Days 1-2)**
1. ✅ Backup original codebase
2. ✅ Create new directory structure
3. ✅ Create DI container
4. ✅ Create configuration system
5. ✅ Migrate shared utilities (utils, logger, errors)

### **Phase 2: Domain Layer (Days 3-4)**
1. ✅ Create domain entities
2. ✅ Define domain interfaces
3. ✅ Migrate business logic
4. ✅ Create domain errors
5. ✅ Domain tests

### **Phase 3: Infrastructure Layer (Days 5-7)**
1. ✅ Migrate providers (Gemini optimization)
2. ✅ Migrate tools system
3. ✅ Migrate repositories
4. ✅ Infrastructure tests

### **Phase 4: Application Layer (Days 8-9)**
1. ✅ Create application services
2. ✅ Implement use cases
3. ✅ Create DTOs
4. ✅ Application tests

### **Phase 5: Presentation Layer (Days 10-11)**
1. ✅ Create HTTP handlers
2. ✅ Implement middleware
3. ✅ Create API controllers
4. ✅ SSE streaming implementation
5. ✅ Integration tests

### **Phase 6: Integration (Day 12)**
1. ✅ Wire everything with DI
2. ✅ Main entry point
3. ✅ End-to-end tests
4. ✅ Performance testing

### **Phase 7: Documentation (Day 13)**
1. ✅ API documentation
2. ✅ Architecture diagrams
3. ✅ Migration guide
4. ✅ Troubleshooting guide

---

## 🔄 Step-by-Step Migration

### **Step 1: Shared Utilities**
```bash
# Migrate existing shared code
✅ internal/utils/          -> internal/utils/
✅ internal/logger/         -> internal/logger/
✅ internal/errors/         -> internal/errors/
✅ internal/cache/          -> internal/cache/
✅ internal/redis/          -> internal/redis/
✅ internal/pubsub/         -> internal/pubsub/
```

### **Step 2: Domain Layer**
```bash
# Create new domain entities
✅ internal/domain/entities/agent.go
✅ internal/domain/entities/conversation.go
✅ internal/domain/entities/tool.go
✅ internal/domain/entities/errors.go
✅ internal/domain/entities/complete.go

# Create domain interfaces
✅ internal/domain/interfaces/agent_service.go
✅ internal/domain/interfaces/llm_provider.go
✅ internal/domain/interfaces/tool_executor.go
✅ internal/domain/interfaces/repository.go
✅ internal/domain/interfaces/orchestration.go
```

### **Step 3: Infrastructure Layer**
```bash
# Migrate providers
✅ internal/infrastructure/providers/gemini_provider.go
✅ internal/infrastructure/providers/openrouter_provider.go
✅ internal/infrastructure/providers/provider_factory.go

# Migrate tools
✅ internal/infrastructure/tools/financial_research.go
✅ internal/infrastructure/tools/financial_scrape.go
✅ internal/infrastructure/tools/financial_calculate.go
✅ internal/infrastructure/tools/export_report.go
✅ internal/infrastructure/tools/tool_registry.go

# Migrate repositories
✅ internal/infrastructure/repositories/conversation_repo.go
✅ internal/infrastructure/repositories/cache_repository.go
```

### **Step 4: Application Layer**
```bash
# Create application services
✅ internal/application/services/agent_service_impl.go
✅ internal/application/services/orchestration_service.go
✅ internal/application/services/tool_execution_service.go

# Create use cases
✅ internal/application/use_cases/chat_use_case.go
✅ internal/application/use_cases/tool_use_case.go
✅ internal/application/use_cases/handoff_use_case.go

# Create DTOs
✅ internal/application/dtos/request_dto.go
✅ internal/application/dtos/response_dto.go
```

### **Step 5: Presentation Layer**
```bash
# Create handlers
✅ internal/presentation/handlers/chat_handler.go
✅ internal/presentation/handlers/stream_handler.go
✅ internal/presentation/handlers/health_handler.go
✅ internal/presentation/handlers/history_handler.go

# Create middleware
✅ internal/presentation/middleware/auth_middleware.go
✅ internal/presentation/middleware/rate_limit_middleware.go
✅ internal/presentation/middleware/cors_middleware.go
✅ internal/presentation/middleware/logging_middleware.go

# Create controllers
✅ internal/presentation/controllers/chat_controller.go
```

### **Step 6: Wiring**
```bash
# DI Container
✅ internal/di/container.go
✅ internal/di/bindings.go
✅ internal/di/providers.go

# Configuration
✅ internal/config/config.go

# Main entry point
✅ cmd/gemini-cli/main.go (new)
```

---

## 🛡️ Rollback Strategy

### **Quick Rollback Commands**:
```bash
# Rollback to original
cd ..
rm -rf Gemini-backup-original
cp -r Gemini-backup-original Gemini
cd Gemini

# Git revert (if using git)
git checkout -- .
git clean -fd
```

### **Rollback Points**:
1. ✅ After Phase 1 (Foundation)
2. ✅ After Phase 2 (Domain Layer)
3. ✅ After Phase 3 (Infrastructure)
4. ✅ After Phase 4 (Application)
5. ✅ After Phase 5 (Presentation)
6. ✅ After Phase 6 (Integration)

---

## 📝 Testing Strategy

### **Unit Tests**:
- Domain layer: Target 80% coverage
- Infrastructure: Target 70% coverage
- Application: Target 75% coverage

### **Integration Tests**:
- API endpoints: All endpoints tested
- Tool execution: All tools tested
- Provider integration: All providers tested

### **Performance Tests**:
- Streaming latency
- Response time
- Memory usage
- Concurrency handling

---

## 🎯 Success Criteria

### **Functionality**:
- ✅ All existing features work
- ✅ No breaking changes to API
- ✅ All tools function correctly
- ✅ SSE streaming works
- ✅ Agent routing works

### **Quality**:
- ✅ No compilation errors
- ✅ All tests pass
- ✅ Performance metrics acceptable
- ✅ No memory leaks

### **Documentation**:
- ✅ API documentation updated
- ✅ Architecture diagrams updated
- ✅ Migration guide complete

---

## ⚠️ Risk Mitigation

### **Risks**:
1. **API Breaking Changes**: Fix by maintaining backward compatibility
2. **Performance Degradation**: Benchmark and optimize
3. **Integration Issues**: Test thoroughly before deployment
4. **Tool Failures**: Keep original implementation as fallback

### **Mitigation**:
- Incremental migration with checkpoints
- Comprehensive testing at each phase
- Performance monitoring
- Rollback plan ready

---

## 📊 Migration Checklist

### **Before Migration**:
- [ ] Backup original codebase ✅
- [ ] Create migration plan ✅
- [ ] Set up testing environment ✅
- [ ] Document current state ✅

### **During Migration**:
- [ ] Migrate shared utilities
- [ ] Create domain layer
- [ ] Migrate infrastructure
- [ ] Create application layer
- [ ] Create presentation layer
- [ ] Wire with DI container
- [ ] Create main entry point

### **After Migration**:
- [ ] Build and test
- [ ] Run all tests
- [ ] Performance benchmark
- [ ] Documentation update
- [ ] Rollback test
- [ ] Deploy to staging

---

## 🎉 Expected Outcomes

### **Code Quality**:
- Better separation of concerns
- Easier to test
- Easier to maintain
- Easier to scale

### **Performance**:
- Better streaming
- Optimized providers
- Better caching
- Reduced latency

### **Developer Experience**:
- Clearer architecture
- Better documentation
- Easier to add features
- Better error handling

---

*Migration Plan Version: 1.0*
*Created: June 16, 2026*
*Owner: Rabuno*