<!-- Generated: 2026-06-18 | Files scanned: 142 | Token estimate: ~850 -->

# Frontend Architecture

## Entry Point
`frontend/src/app/app.ts` - Bootstraps the FSD architecture with chat page initialization

## Page Architecture

### Main Page
```
chat/page.ts → createChatPage()
    ├── sidebar/conversation-list.ts
    ├── chat-view/compose.ts
    └── pipeline/compose.ts
```

## State Management (Nanostores)

### Chat State (`entities/chat/model/store.ts`)
```typescript
$currentChatId - Current active session ID
$currentChatTitle - Session title
$isGenerating - Loading state
$pipeline - Real-time execution pipeline state
$hasActiveChat - Computed: has active session
```

### Session State (`entities/session/model/store.ts`)
- Session CRUD operations
- Session persistence via API

## Component Hierarchy

### UI Components (`shared/ui/`)
- `icon-button.ts` - Reusable icon button with hover effects
- `skeleton.ts` - Loading skeleton placeholders
- `toast.ts` - Toast notification system
- `connection-status.ts` - SSE connection indicator

### Feature Components (`features/`)
```
chat/send/
├── ui.ts - Chat input with file upload
└── model.ts - Send message logic

sidebar/toggle/
├── ui.ts - Sidebar collapse/expand button
└── model.ts - Sidebar state management

theme/toggle/
├── ui.ts - Dark/light mode toggle
└── model.ts - Theme state

settings/modal/
├── ui.ts - Settings modal UI
└── model.ts - Settings management
```

### Entity Components (`entities/`)
```
chat/
├── api/history.ts - Chat history API client
└── model/types.ts - Chat message types

session/
├── api/crud.ts - Session CRUD operations
└── model/types.ts - Session types
```

## Data Flow

### SSE Event System
```
Backend SSE → shared/api/sse.ts → Event parsing → State updates → UI re-render
Event types: token, tool_call, tool_result, agent_switch, done, error
```

### API Flow
```
UI Component → shared/api/client.ts → API endpoint → Response → State update
```

## Key Modules

### API Layer (`shared/api/`)
- `client.ts` - HTTP client with interceptors
- `sse.ts` - SSE connection manager
- `types.ts` - API type definitions
- `mock-news.ts` - Development mock data

### Utilities (`shared/lib/`)
- `dom.ts` - DOM manipulation helpers
- `html.ts` - HTML sanitization
- `markdown.ts` - Markdown rendering with Prism.js
- `errors.ts` - Error handling utilities

### Widget Layer (`widgets/`)
- `chat-view/compose.ts` - Chat view composition
- `sidebar/compose.ts` - Sidebar composition
- `pipeline/compose.ts` - Execution pipeline visualization

## Build System
- **Vite** - Fast dev server and production builds
- **TypeScript** - Strict mode with full type safety
- **Nanostores** - Reactive state management
- **Prism.js** - Syntax highlighting
- **Marked** - Markdown parsing

## Page Structure
```
App Layout
├── Sidebar (Session List)
├── Main Chat Area
│   ├── Message History
│   ├── Input Area
│   └── Pipeline Status
└── Settings Modal
```

## Technology Stack
- Vanilla TypeScript (no framework)
- Nanostores for reactive state
- CSS custom properties for theming
- Glassmorphic design system
- Mobile-responsive layout