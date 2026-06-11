# ADR-002: ReAct Loop Architecture

> Status: Accepted | Date: 2026-06-12

## Context

The AI agent needs to reason about financial queries, dispatch tools for real-time data, and iterate until a final answer is produced.

## Decision

Implement a ReAct (Reasoning + Acting) loop:
1. Build context (route + skills + real-time market data)
2. Send condensed history to LLM
3. If LLM returns tool calls, execute them, append results, loop back
4. If LLM returns text, return as final response
5. Max 2-3 tool calls per turn to prevent infinite loops

## Consequences

- Multi-step reasoning capability
- Tool call overhead adds latency per iteration
- LRU cache on search/scrape results reduces quota consumption
