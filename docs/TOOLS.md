# Tool Reference

> 6 tools available to the AI agent, caching strategy, and security measures.

## Available Tools

| Tool | Description | Data Source |
|------|-------------|-------------|
| `financial_research` | Google Search via SerpAPI | SerpAPI (Google Search, Vietnamese locale) |
| `tavily_search` | Tavily Search (AI-powered) | Tavily API (advanced depth, direct answers) |
| `financial_scrape` | Deep web content extraction | HTTP GET + regex extraction (SSRF protected) |
| `financial_calculate` | Financial calculations | Stub (returns formatted string) |
| `handoff_request` | Delegate to another agent | Internal handoff mechanism |
| `load_financial_context` | Load agent/skill documents | GitHub raw content API (cached) |

## Tool Details

### `financial_research`

Searches Google via SerpAPI with Vietnamese locale (`gl=vn`). Extracts answer boxes, stock information, and top 3 organic results.

**Caching:** Results cached by normalized query key with `google_` prefix. LRU eviction at 200 entries.

### `tavily_search`

POST to Tavily API with `search_depth: "advanced"` and `include_answer: true`. Extracts AI-generated direct answer and top 5 results.

**Caching:** Results cached by normalized query key with `tavily_` prefix. LRU eviction at 200 entries.

### `financial_scrape`

Fetches a URL and extracts text from `<p>`, `<h1>`-`<h3>`, `<li>` tags. Filters short text (< 20 chars), truncates to 8000 chars.

**SSRF Protection:** `isBlockedURL()` validates:
- Only `http`/`https` schemes
- Blocks loopback, link-local, private ranges (`10.x`, `172.16-31.x`, `192.168.x`)
- Blocks IPv6 loopback and link-local

**Caching:** Results cached by normalized URL key with `scrape_` prefix. LRU eviction at 200 entries.

### `financial_calculate`

**Stub implementation** — returns a formatted string rather than evaluating expressions.

### `handoff_request`

Sets `agent.handoffPlan`. The orchestrator picks it up on the next ReAct loop iteration.

**Parameters:** `target_agent` (required), `reason` (required), `task_payload` (optional)

### `load_financial_context`

Loads agent/skill markdown from GitHub. Results cached in-memory (never evicted).

**URL patterns:**
- Agent: `{repo}/agent-plugins/{slug}/agents/{slug}.md`
- Skill: `{repo}/agent-plugins/{agent}/skills/{skill}/SKILL.md`

## Caching Strategy

| Cache | Type | Max Entries | Key Format |
|-------|------|-------------|------------|
| Search/Scrape | LRU | 200 | `{prefix}_{normalized_query}` |
| Documents | In-memory | Unlimited | `{docType}:{name}` |

All caches are thread-safe (`sync.RWMutex`).

## Real-Time Data Detection

`NeedsRealtimeData()` checks for time-sensitive keywords in Vietnamese and English ("hom nay", "stock price", "realtime", etc.). When detected, Google and Tavily are fetched in parallel via `sync.WaitGroup`.
