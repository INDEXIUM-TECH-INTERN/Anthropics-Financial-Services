# ADR-004: AI-Powered Routing with Heuristic Fallback

> Status: Accepted | Date: 2026-06-12

## Context

10 specialist financial agents exist. Selecting the right one for each query is critical for response quality.

## Decision

Two-tier routing:
- **Tier 1**: AI-powered router sends query + agent catalog to LLM, parses JSON RoutePlan
- **Tier 2**: Vietnamese keyword-matching heuristic fallback (10 keywords → agents)
- **Sanitization**: validates agent whitelist, verifies GitHub docs via HTTP HEAD, filters valid skills
- Also detects temporal intent ("hom nay", "ngay mai", "nam 2024") for time-sensitive queries

## Consequences

- Natural language queries reach the correct specialist
- AI router costs one LLM call per new conversation
- Heuristic fallback is free but less accurate
- 40+ slash commands bypass routing for common workflows
