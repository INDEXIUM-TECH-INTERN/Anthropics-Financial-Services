# Frontend — Vanilla TypeScript SPA

<!-- Generated: 2026-06-14 | TS files: 23 | Token estimate: ~900 -->

## Entry Point
`frontend/src/main.ts` — app bootstrap, 37KB. Initializes all modules and mounts the UI.

## Architecture
No framework — vanilla TypeScript with modular ES modules loaded via Vite.

```
main.ts (bootstrap)
├── components/chat/
│   ├── message-bubble.ts      — Renders user/assistant messages, markdown
│   ├── code-block.ts          — Syntax-highlighted code blocks
│   ├── thinking-card.ts       — Tool-call thinking/reasoning display
│   ├── streaming-cursor.ts    — Animated SSE streaming cursor
│   └── typing-indicator.ts    — Typing animation
│
├── components/sidebar/
│   └── conversation-list.ts   — Session list, create/rename/delete
│
├── components/ui/
│   ├── icon-button.ts         — Reusable icon button
│   ├── skeleton.ts            — Loading skeleton placeholder
│   ├── toast.ts               — Toast notification system
│   └── connection-status.ts   — SSE connection indicator
│
├── services/
│   ├── api.ts                 — REST API calls (chat, history, sessions, config)
│   ├── sse-manager.ts         — SSE connection + event parsing + callbacks
│   ├── markdown.ts            — Markdown → HTML renderer
│   ├── error-handler.ts       — Global error handling + user feedback
│   └── connection-status.ts   — Connection state tracking
│
├── stores/
│   └── app-state.ts           — Global reactive state (current chat, messages, settings)
│
├── hooks/
│   ├── useAutoResize.ts       — Auto-resize textarea hook
│   └── useScrollToBottom.ts   — Auto-scroll chat to bottom
│
├── types/
│   └── api.ts                 — TypeScript interfaces for API types
│
├── utils/
│   └── dom.ts                 — DOM helper utilities
│
└── styles/
    ├── main.css               — Entry stylesheet
    ├── design-tokens.css      — CSS custom properties (colors, spacing)
    ├── base.css               — Reset + base styles
    ├── glass.css              — Glassmorphic effects
    ├── layout.css             — App layout (sidebar + main)
    ├── chat.css               — Chat area styles
    ├── components.css         — Shared component styles
    ├── input.css              — Input/textarea styles
    ├── modal.css              — Modal dialog styles
    ├── pipeline.css           — Execution pipeline visualization
    └── animations.css         — Keyframe animations
```

## State Management
`stores/app-state.ts` — Simple reactive store pattern:
- `state`: messages[], currentChatId, settings, connection status
- `subscribe(listener)`: register state change listeners
- `setState(partial)`: update state and notify subscribers
- Reducer-like actions for message append, session switch, settings update

## SSE Event Flow
```
sse-manager.ts connects to GET /api/events?chat_id=xxx
    │
    ├── Event types:
    │   ├── "token"        → append text to current message bubble
    │   ├── "tool_call"    → render thinking-card
    │   ├── "tool_result"  → update thinking-card with result
    │   ├── "agent_switch" → show agent indicator
    │   ├── "done"         → finalize message, update history
    │   └── "error"        → show toast notification
    │
    └── Event data streamed via sse-manager → app-state → components re-render
```

## Build System
- **Vite** (`vite.config.ts`) — dev server + production build
- Output: `frontend/dist/`
- TypeScript: `tsconfig.json` (strict mode) + `tsconfig.node.json` (Vite config)

## API Client (`services/api.ts`)
```typescript
POST /api/chat          → sync chat
POST /api/chat/stream   → streaming chat (returns SSE event source)
GET  /api/history       → get conversation history
GET  /api/chats         → list all sessions
POST /api/reset         → reset session
PUT  /api/config/keys   → update API keys (requires X-Config-Secret)
```

## UI Pattern
- Glassmorphic design with CSS custom properties (design-tokens.css)
- Responsive: sidebar collapses on mobile
- Vietnamese-first UI text
- No external UI library — all components hand-rolled
