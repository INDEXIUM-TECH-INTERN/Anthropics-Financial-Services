# All-in-one image for Render Free tier: Go API + built frontend SPA

# ---- Frontend build ----
FROM node:22-alpine AS frontend-builder

WORKDIR /app/frontend

COPY frontend/package.json frontend/package-lock.json* ./
RUN npm ci --ignore-scripts 2>/dev/null || npm install --ignore-scripts

COPY frontend/ ./
RUN npm run build

# ---- Go build ----
FROM golang:1.25-alpine AS go-builder

WORKDIR /app

COPY Gemini/go.mod Gemini/go.sum ./
RUN go mod download

COPY Gemini/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /gemini-cli ./cmd/gemini-cli

# ---- Runtime ----
FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=go-builder /gemini-cli .
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

RUN adduser -D -g '' appuser
USER appuser

EXPOSE 8080

ENV PORT=8080

ENTRYPOINT ["./gemini-cli"]
CMD ["--server"]