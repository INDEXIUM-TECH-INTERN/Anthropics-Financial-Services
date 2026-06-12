# ADR-006: SSRF Protection for Tool Execution

## Status

Accepted

## Context

The AI agent can invoke tools that fetch external URLs (`financial_scrape`, `read_local_file`). Without validation, a malicious or careless AI response could cause the server to fetch internal network resources (SSRF — Server-Side Request Forgery), such as:
- `http://localhost:6379` (Redis)
- `http://169.254.169.254` (cloud metadata)
- Internal microservice endpoints

## Decision

Implement **layered SSRF protection** at multiple levels:

### Layer 1: URL Allowlist (Tool Schema)
The `read_local_file` tool uses an **extension allowlist**:
```go
allowedExts := map[string]bool{
    ".xlsx": true, ".csv": true, ".txt": true,
    ".md": true, ".json": true, ".pdf": true,
    ".docx": true, ".pptx": true,
}
```
Only files with allowed extensions in the `exports/` directory can be read.

### Layer 2: Path Traversal Prevention
```go
baseName := filepath.Base(path)
if baseName == "" || baseName == "." || baseName == ".." {
    return "Error: Invalid filename"
}
```
The resolved absolute path must start with the allowed directory's absolute path.

### Layer 3: Search Directory Restriction
`read_local_file` only searches within predefined directories:
```go
searchDirs := []string{
    "frontend/exports",
    "Gemini/frontend/exports",
    "../frontend/exports",
    "exports",
}
```

### Layer 4: `financial_scrape` URL Validation (Future)
- Block private IP ranges (10.x, 172.16-31.x, 192.168.x, 127.x)
- Block link-local (169.254.x)
- Only allow `http://` and `https://` schemes

## Consequences

### Positive
- Path traversal attacks blocked by design.
- Only exported reports can be read back by the AI.
- Extension allowlist prevents unexpected file types.

### Negative
- `financial_scrape` has **no URL validation yet** — relies on AI not generating malicious URLs.

## Future Work

- [ ] Add URL validation to `tools.ScrapeWeb(url)` for SSRF hardening
- [ ] Add `DISABLE_EXTERNAL_SCRAPE` env var for air-gapped deployments

## Related

- `Gemini/internal/core/dispatcher.go` — `handleReadLocalFile`, `handleFinancialScrape`
- ADR-001: Multi-Provider Failover
