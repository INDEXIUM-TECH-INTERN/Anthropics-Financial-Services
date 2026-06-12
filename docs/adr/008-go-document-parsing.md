# ADR-008: Go Document Parsing for Financial Files

## Status

Accepted

## Context

Financial analysts upload various document types (PDF reports, Excel spreadsheets, CSV data) for AI analysis. The backend needs to extract text content from these files efficiently without spawning external processes.

## Decision

Use **pure Go libraries** for document parsing, with a unified `ParseAttachment()` interface:

### Supported Formats

| Format | Library | Notes |
|--------|---------|-------|
| PDF | `pdfcpu` or `rsc.io/pdf` | Text extraction, page-by-page |
| XLSX | `excelize` or `xuri/excelize/v2` | Full Excel support with formulas |
| CSV | `encoding/csv` | Standard library |
| DOCX | `unioffice` or custom XML parse | Word document extraction |
| PPTX | `unioffice` | PowerPoint extraction |
| TXT/MD | `os.ReadFile` | Direct read |
| JSON/XML | `encoding/json`, `encoding/xml` | Standard library |

### Interface Design

```go
// ParseAttachment extracts text content from a file.
// Returns (content, true) on success, ("", false) on unsupported type.
func ParseAttachment(name, mimeType, dataB64 string) (string, bool)
```

### Flow

```
User uploads file
  → appendUserTextInternal()
    → utils.ParseAttachment(name, mimeType, dataB64)
      → Decode base64
      → Detect type by extension + mime
      → Extract text via format-specific parser
      → Return extracted text
    → Append to message content
  → AI receives file content as text in conversation
```

### Size Limits

- Files are read into memory — enforce `MAX_UPLOAD_SIZE` (default: 10MB).
- Extracted text is truncated to `MAX_FILE_CONTENT_CHARS` (default: 50,000 chars) before being added to conversation context.

## Consequences

### Positive
- No external process spawning (Python, LibreOffice) — pure Go.
- Unified interface makes adding new formats straightforward.
- Base64 encoding allows binary files to travel through JSON APIs.

### Negative
- Go PDF parsing is less mature than Python's `pdfplumber`/`PyPDF2`.
- Large Excel files may consume significant memory.
- No OCR support for scanned documents (would require external service).

## Alternatives Considered

| Approach | Why Rejected |
|----------|-------------|
| Python subprocess (`markitdown`) | Adds Python dependency, slower, harder to deploy |
| External API (AWS Textract) | Cost, latency, vendor lock-in |
| `libreoffice --headless` | Requires LibreOffice installation, fragile in containers |

## Related

- `Gemini/internal/utils/` — `ParseAttachment`, `GetFileContentWrapper`
- `Gemini/internal/core/agent.go` — `appendUserTextInternal` integration
- ADR-005: Context Summarization (large file content feeds into context window)
