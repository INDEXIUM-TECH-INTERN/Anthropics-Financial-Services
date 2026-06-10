# Domain Glossary

> Key concepts and terminology used throughout the project.

## Core Architecture

### Message
A provider-agnostic container for conversation data. Encapsulates the role (`system`, `user`, `assistant`, `tool`), text content, and structured data for tool calls or responses. Defined in `internal/models/messaging/`.

### Provider
An interface that abstracts an LLM vendor. Translates neutral `Message` objects and `ToolSchema` definitions into vendor-specific API calls. Implemented by `GeminiProvider`, `OpenRouterProvider`, and `MultiProvider`.

### ContextWindow
The central repository for conversation history. Maintains the "source of truth" for a dialogue as a sequence of neutral `Message` objects. Handles summarization when the token limit is exceeded. Defined in `internal/core/context_window.go`.

### ToolSchema
A neutral definition of a function or capability that an agent can invoke. Describes the name, purpose, and required parameters in a format that adapters can translate for specific LLMs.

### Orchestrator
The flow-control module responsible for the main ReAct loop: receiving input, updating memory, calling providers, and dispatching tool results. Defined in `internal/core/orchestrator.go`.

### Agent
The central struct that wires together all subsystems: provider (LLM), orchestrator (ReAct loop), dispatcher (tool execution), conversation/context window, and system prompt. Defined in `internal/core/agent.go`.

### RoutePlan
A structured routing decision containing the target agent, skills, temporal intent, and reason. Produced by the AI router or heuristic fallback. Defined in `internal/core/routing.go`.

### Dispatcher
The "action" half of the ReAct loop. Defines available tools as JSON schemas for the LLM and handles execution of tool calls. Includes LRU caching for search results. Defined in `internal/core/dispatcher.go`.

## Workflow Concepts

### ReAct Loop
**Reasoning + Acting** — the core interaction pattern. The LLM reasons about the task, calls tools to gather information, processes the results, and reasons again until it can produce a final response.

### Bootstrap Context
The initial context loaded at the start of a new conversation: agent configuration, agent markdown document, skill markdown documents, and optionally real-time market data.

### Handoff
A mechanism for the AI to delegate to a different specialist agent mid-conversation. The orchestrator picks up the handoff plan and loads the new agent's context on the next loop iteration.

### Slash Commands
Predefined commands that bypass routing: `/earnings` -> `earnings-reviewer`, `/market` -> `market-researcher`.

### Greeting Fast-Path
Short inputs (< 5 chars) and casual greetings skip the expensive AI routing step and enter the ReAct loop directly. Uses Vietnamese accent normalization for robust matching.

## Temporal Concepts

### Temporal Intent
The time context extracted from a user query:
- `realtime` — current/live data ("hom nay", "hien tai")
- `latest` — most recent available ("hom qua", "gan day")
- `historical` — past data with specific dates ("nam 2024", "quy 1")
- `is_future` — future predictions ("ngay mai", "du bao")

## Provider Concepts

### MultiProvider
A wrapper around a primary provider and multiple fallback providers. Handles automatic failover, quota detection, rate-limit avoidance, and gradual recovery. Defined in `providers/multiprovider.go`.

### Quota-Aware Skipping
When a quota/rate-limit error is detected, the primary provider is skipped for an increasing number of subsequent calls to prevent spamming.

### Gradual Recovery
When fallback providers succeed, the skip counter is halved, allowing the primary to be tried again gradually.

## Caching Concepts

### LRU Cache
Least Recently Used eviction policy for the search/scrape result cache. Max 200 entries, oldest removed first. Used in the Dispatcher.

### Document Cache
In-memory cache for agent/skill markdown documents loaded from GitHub. Keyed by `{docType}:{name}`, never evicted. Used in the Registry.

## Storage Concepts

### ChatSession
A persistent conversation container with ID, title, message list, and timestamp. Stored in Redis with in-memory fallback. Defined in `internal/store/session_store.go`.

### Conversation
A lightweight state container holding the current `ContextWindow`. The `Agent` holds the conversation and orchestrator modifies it during the ReAct loop.

## Security Concepts

### SSRF Protection
Server-Side Request Forgery protection in the web scraper (`tools/scraper/scraper.go`). Validates URLs against private/internal IP ranges before making HTTP requests.

### Request Body Limit
1MB maximum on the `/api/chat` endpoint to prevent DoS attacks via `http.MaxBytesReader`.

### CORS
Cross-Origin Resource Sharing headers on all API endpoints. Currently allows all origins (`*`) — should be restricted in production.

### URL Validation
The scraper validates URL schemes (only `http`/`https`) and resolves hostnames to check against blocked IP ranges (loopback, link-local, private ranges).
