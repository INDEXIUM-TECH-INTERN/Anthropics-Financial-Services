# ADR-005: Context Window Summarization Strategy

## Status

Accepted

## Context

LLM providers have finite context windows (128k–1M tokens). Long-running financial research conversations accumulate messages rapidly (user queries, tool calls, tool responses, AI reasoning). Without mitigation, the context window fills up and the LLM either fails or loses critical early instructions.

## Decision

Implement **selective summarization** that:
1. **Preserves full history** in memory for UI display and session serialization.
2. **Builds a condensed view** for LLM consumption via `BuildLLMHistory()`:
   - MemorySummary (if exists) is always inserted first.
   - The first 2 messages (user query + agent/skill bootstrap) are always protected.
   - Only the last N (default 7) messages are included verbatim.
   - Middle messages are replaced by the summary.
3. Triggers summarization when `EstimateCurrentTokens() > CONTEXT_MAX_TOKENS` (default 92,000).

### Summarization Flow

```
ShouldSummarize() → true
  → SummarizeOldest(provider, keepRecent, maxSummaryChars)
    → Extract messages[:len-history-keepRecent]
    → Format with role labels + tool call summaries
    → Truncate to maxSummaryInputChars (default 18,000 chars)
    → Call provider.GenerateText() with summarization prompt
    → UpdateSummary(summary, endIdx)
```

### Configuration (Environment Variables)

| Variable | Default | Purpose |
|----------|---------|---------|
| `CONTEXT_MAX_TOKENS` | 92000 | Token threshold to trigger summarization |
| `CONTEXT_KEEP_RECENT` | 7 | Number of recent messages to preserve verbatim |
| `CONTEXT_MAX_SUMMARY_INPUT` | 18000 | Max chars fed to summarization LLM call |

## Consequences

### Positive
- Conversations can run indefinitely without context overflow.
- Full history preserved for UI/debugging.
- Configurable thresholds for different model context windows.
- Bootstrap instructions never lost (protected first 2 messages).

### Negative
- Summarization costs extra API tokens (one LLM call per trigger).
- Historical nuances may be lost in summary.
- Requires `context_summarizer.txt` prompt template.

## Alternatives Considered

| Approach | Why Rejected |
|----------|-------------|
| Trim oldest messages | Loses bootstrap instructions and important context |
| Fixed-size sliding window | Same problem — loses early instructions |
| External vector store | Overkill for this use case; adds infrastructure dependency |
| Summarize every N messages | Wasteful when conversation is short |

## Related

- `Gemini/internal/core/context_window.go` — Core implementation
- `Gemini/internal/core/context_window_test.go` — 10 test cases
- ADR-002: ReAct Loop Architecture (summarization integrates with the loop)
