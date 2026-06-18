.PHONY: build test test-race test-cover test-verbose clean run fmt lint

# Run tests (scoped to internal packages — stays in Gemini dir)
test:
	go test ./internal/... ./cmd/... ./api/...

# Run tests with race detector
test-race:
	go test -race ./internal/...

# Run tests with coverage
test-cover:
	go test -cover -coverprofile=coverage.out ./internal/...
	go tool cover -func=coverage.out

# Run tests with verbose output
test-verbose:
	go test -v ./internal/...

# Run tests for specific package (usage: make test-pkg PKG=core)
test-pkg:
	go test -v ./internal/$(PKG)/...

# Build the binary
build:
	go build -o gemini ./cmd/gemini-cli

# Clean build artifacts
clean:
	rm -f gemini gemini.exe coverage.out

# Run the server
server:
	./gemini -server

# Lint
lint:
	go vet ./...

# Format
fmt:
	goimports -w .
	gofmt -w .
