# LLM Providers & Context Management

> Multi-provider failover, context window management, and session storage.

## Provider Interface

```go
type Provider interface {
    GenerateText(systemPrompt, userPrompt string) (string, error)
    Generate(ctx context.Context, req messaging.Request) (messaging.Message, error)
}
```

## Providers

| Provider | Description | Default Model |
|----------|-------------|---------------|
| `GeminiProvider` | Google Generative Language API | `gemini-flash-latest` |
| `OpenRouterProvider` | OpenAI-compatible API | `meta-llama/llama-3.3-70b-instruct:free` |
| `MultiProvider` | Failover wrapper | Primary + fallbacks |

## Priority Order

1. **Gemini** (primary) — if `GEMINI_API_KEY` is set
2. **OpenRouter Key #1** (fallback)
3. **OpenRouter Key #2** (fallback)
4. **OpenRouter Key #3** (fallback)

If `USE_OPENROUTER_ONLY=1` or no Gemini key, OpenRouter becomes primary.

## Free Model Chain (OpenRouter)

When a model ends with `:free`, the system auto-tries:
1. `nvidia/nemotron-3-super-120b-a12b:free`
2. `google/gemini-2.0-flash-exp:free`
3. `meta-llama/llama-3.3-70b-instruct:free`
4. `mistralai/mistral-7b-instruct:free`
5. `qwen/qwen-2.5-7b-instruct:free`

## Multi-Provider Failover

### Quota-Aware Skipping

On quota/rate-limit errors (detects "quota", "rate", "429", "exceeded"):
- `skipPrimaryUntil` = 5–12 (increases with consecutive failures)
- Primary is skipped for that many subsequent calls

### Gradual Recovery

When a fallback succeeds, `skipPrimaryUntil` is halved, allowing the primary to be retried gradually.

### Exponential Backoff

```
delay = 500ms x 2^attempt + random_jitter(0-300ms), capped at 5s
```

### Round-Robin

Fallback providers rotate via `currentIdx`.

## Context Window Management

### Principles

- **Full history is NEVER deleted** — kept intact for UI display
- **Summarization only affects** what is sent to the LLM

### Summarization Process

When token limit is exceeded:
1. Keep first 2 messages (query + bootstrap) + last N messages
2. Send middle messages to LLM with summarization prompt
3. Store result in `MemorySummary`
4. `BuildLLMHistory()` -> `[MemorySummary] + [Bootstrap] + [Last N]`

### Token Estimation

Heuristic for Vietnamese + English mixed text:
- `byChar = charCount / 3.4`
- `byWord = wordCount x 1.4`
- Takes maximum of both + structural overhead

### Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `CONTEXT_KEEP_RECENT` | 7 | Recent messages kept intact |
| `CONTEXT_MAX_TOKENS` | 92000 | Token threshold for summarization |
| `CONTEXT_MAX_SUMMARY_INPUT` | 18000 | Max chars for summarization prompt |

## Session Storage

### Redis (Primary)

- Keys: `chat:session:<id>` (data), `chat:sessions:list` (index)
- Format: JSON serialization

### In-Memory Fallback

- Thread-safe (`sync.RWMutex`)
- Data lost on restart

### ChatSession Structure

```go
type ChatSession struct {
    ID        string
    Title     string
    Messages  []messaging.Message
    UpdatedAt time.Time
}
```
