# ADR-001: Multi-Provider Failover with Quota Awareness

> Status: Accepted | Date: 2026-06-12

## Context

The financial AI agent relies on multiple LLM providers (Google Gemini as primary, OpenRouter as fallback). Free-tier quotas can be exhausted within minutes under load.

## Decision

Implement a `MultiProvider` wrapper with:
- Round-robin fallback chain across up to 5 API keys per provider
- Quota-aware primary skipping: on rate-limit error, skip primary for 5-12 subsequent calls
- Exponential backoff with jitter: `500ms * 2^attempt + random(0-300ms)`, capped at 5s
- Gradual recovery: on fallback success, halve the skip counter

## Consequences

- Resilient to individual provider outages
- Gracefully degrades rather than failing
- Slightly higher latency during fallback (backoff delays)
