.PHONY: build test test-race test-cover test-verbose clean run

# Build the binary
build:
	go build -o gemini ./cmd/gemini-cli

# Run tests
test:
	go test ./...

# Run tests with race detector
test-race:
	go test -race ./...

# Run tests with coverage
test-cover:
	go test -cover -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

# Run tests with verbose output
test-verbose:
	go test -v ./...

# Run tests for specific package
test-pkg:
	go test -v ./internal/$(PKG)/...

# Clean build artifacts
clean:
	rm -f gemini gemini.exe coverage.out

# Run the server
run: build
	./gemini -server

# Lint
lint:
	go vet ./...
	staticcheck ./...

# Format
fmt:
	goimports -w .
	gofmt -w .
