# ADR-003: Context Window Summarization

> Status: Accepted | Date: 2026-06-12

## Context

Full conversation history grows unbounded, exceeding LLM token limits (e.g. 128K tokens). Deleting messages loses important context for the UI.

## Decision

Keep full history in memory for UI display. For LLM consumption:
- Estimate token count before each LLM call
- When exceeding threshold (default 92K tokens), use LLM to summarize oldest messages
- Build LLM input as: `[MemorySummary] + [First 2 bootstrap messages] + [Last N messages]`
- Configurable via `CONTEXT_KEEP_RECENT` (default 7) and `CONTEXT_MAX_TOKENS` (default 92000)

## Consequences

- Unlimited conversation length
- Small context window footprint (~summary + bootstrap + recent)
- Summarization adds one extra LLM call per threshold breach
