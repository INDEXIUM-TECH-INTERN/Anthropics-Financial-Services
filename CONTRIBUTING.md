# Contributing to Indexium Financial AI Agent

## Code Standards

- **Language**: Go 1.25+, Vietnamese for user-facing strings, English for code
- **Formatting**: `gofmt` + `goimports` (run `make fmt`)
- **Linting**: `go vet ./...` (run `make lint`)
- **Testing**: All new features require table-driven tests

## Architecture Principles

1. **Immutability**: Create new objects, never mutate existing ones (except conversation history which must append)
2. **Small interfaces**: Provider interface has 3 methods — keep it minimal
3. **Constructor injection**: Use `NewXxx()` functions with dependencies as parameters
4. **Error wrapping**: Always use `fmt.Errorf("context: %w", err)` for error propagation
5. **Thread safety**: Agent uses single `sync.RWMutex` — never nest locks across packages

## Commit Convention

```
feat: add new agent for X
fix: resolve deadlock in Y
refactor: split Z into focused modules
test: add tests for W
docs: update ADR for V
chore: update dependencies
```

## File Size Limits

- Functions: < 50 lines
- Files: < 400 lines (if larger, extract a module)
- Packages: cohesive responsibility (one concern per package)

## Testing Requirements

- Table-driven tests preferred
- Use `MockProvider` from `internal/providers/mock.go` for unit tests
- Run `make test` before every commit
