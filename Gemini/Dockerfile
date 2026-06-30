# ---- Build stage ----
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy module files first (layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /gemini-cli ./cmd/gemini-cli

# ---- Runtime stage ----
FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary and prompt templates from builder
COPY --from=builder /gemini-cli .
COPY --from=builder /app/internal/prompt ./internal/prompt

# Create non-root user
RUN adduser -D -g '' appuser
USER appuser

EXPOSE 8080

ENV PORT=8080

# Default: run server mode
ENTRYPOINT ["./gemini-cli"]
CMD ["--server"]
