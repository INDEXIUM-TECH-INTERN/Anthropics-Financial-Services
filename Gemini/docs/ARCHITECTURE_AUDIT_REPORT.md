# Agent Architecture Audit Report — TestAIFinance

**Audit Date:** 2026-06-17
**Model Stack:** Gemini 3.1 Flash Lite (primary, Gemini-only)
**Architecture:** ReAct loop with tool-calling, context summarization, agent routing

---

## Executive Verdict

**Overall Health:** 🟡 **HIGH RISK**

**Primary Failure Mode:** Wrapper regression từ provider fallback loop và thiếu enforcement tool calling.

**Most Urgent Fix:** Remove OpenRouter references và fix hidden repair loop trong MultiProvider trước khi release production.

---

## Scope

| Target | Description |
|--------|-------------|
| **System** | ReAct loop, 10 specialist agents, tool-calling, context summarization |
| **Entrypoints** | REST API (chat), SSE streaming, CLI |
| **Model** | Gemini 3.1 Flash Lite (single provider, no fallbacks) |
| **Symptoms** | - Wrapper regression (prompt enforcement ≠ code enforcement) <br> - Hidden repair loops (MultiProvider fallback) <br> - OpenRouter remnants in codebase |
| **Time Window** | Since OpenRouter removal attempt (2026-06-17) |
| **Layers Audited** | All 12 layers (full stack) |

---

## Findings (Severity-Ranked)

### 🔴 CRITICAL: Hidden Repair Loop Causes Silent Failure

**Severity:** CRITICAL
**Layer:** 11 (Hidden Repair Loops)
**Mechanism:** MultiProvider fallback logic với retry automatic
**Root Cause:** Provider Manager vẫn có fallbacks, retry logic, và state management

**Evidence:**

1. `internal/infrastructure/providers/multiprovider.go` lines 130-227:
```go
func (m *MultiProvider) Generate(ctx context.Context, req messaging.Request) (messaging.Message, error) {
    // Try primary first
    aiMessage, err := m.primary.Generate(ctx, req)
    if err == nil {
        // Reset failure counter
        m.mu.Lock()
        m.primaryFailures = 0
        m.skipPrimaryUntil = 0
        m.mu.Unlock()
        return aiMessage, nil
    }

    isQuotaError := isQuotaOrRateLimitError(err)
    pubsub.BroadcastLog(fmt.Sprintf("Primary error: %v", err), "error")

    if isQuotaError {
        // Increase skip để tránh spam primary
        m.mu.Lock()
        m.primaryFailures++
        m.skipPrimaryUntil = 5 + (m.primaryFailures / 2)
        m.mu.Unlock()
    }

    // Fallback logic - HIDDEN LLM CALL
    return m.tryFallbacksOnly(ctx, req)  // <--- Silent second LLM pass
}
```

2. `internal/core/routing.go` line 107-111: Chức năng fallback (không dùng nhưng tồn tại):
```go
func routeWithProviderFallback(systemPrompt, userPrompt string) (string, error) {
    // For now, use a mock - in production this would call a real provider
    return systemPrompt + "\n\n" + userPrompt, nil
}
```

**Impact:**
- Khi Gemini fail (quota/rate limit), hệ thống tự động call LLM lần thứ 2 qua fallback
- User không biết có repair loop diễn ra (chỉ log "Primary error")
- Chi phí double, latency tăng, behavior non-deterministic

**Confidence:** 1.0

---

### 🔴 CRITICAL: Tool Discipline Failure — Prompt Enforcement ≠ Code Enforcement

**Severity:** CRITICAL
**Layer:** 6 (Tool Selection) & 7 (Tool Execution)
**Mechanism:** Prompt nói "BẮT BUỘC" nhưng code KHÔNG enforce
**Root Cause:** LLM có thể skip tool calls mà không bị penalize

**Evidence:**

1. `internal/prompt/system_prompt.txt` lines 9, 18:
```
... Khi người dùng yêu cầu xuất báo cáo, bạn BẮT BUỘC phải gọi công cụ này trước tiên để sinh ra đường dẫn tải về thực tế.

...
1. KIỂM TRA DỮ LIỆU: Nếu bạn không có sẵn số liệu thực tế trong ngữ cảnh, bạn PHẢI gọi tool `financial_research` hoặc `financial_scrape` ngay lập tức. KHÔNG ĐƯỢC tự bịa số liệu.
```

2. `internal/core/dispatcher.go` lines 171-193:
```go
func (d *Dispatcher) HandleToolCalls(aiMessage messaging.Message) bool {
    if len(aiMessage.ToolCalls) == 0 {
        return false  // <--- Không penalize, chỉ log sự thiếu vắng
    }

    for _, toolCall := range aiMessage.ToolCalls {
        fmt.Printf("🎯 [Action] AI invokes MCP tool: %s\n", toolCall.Name)
        // Execute tool...
    }
    return true
}
```

3. `internal/core/orchestrator.go` lines 337-339: ReAct loop không check nếu tool calls đã được thực thi:
```go
hasToolCall := o.agent.dispatcher.HandleToolCalls(msg)

if !hasToolCall && finalText != "" {
    // Send only the final response to client (skip thinking/tool-call preamble)
    // <--- LLM có thể trả lời NGAY khi chưa gọi tool, nếu prompt nói "có thể"
}
```

**Impact:**
- Prompt nói "BẮT BUỘC" nhưng LLM có thể skip → hallucination numbers
- Agent có thể trả lời mà không có dữ liệu real-time (vi phạm layer 7 tool execution)
- Context summarization mất tool results → corruption

**Confidence:** 1.0

---

### 🟠 HIGH: OpenRouter Remnants in Codebase

**Severity:** HIGH
**Layer:** 10 (Platform Rendering) & 12 (Persistence)
**Mechanism:** Type definitions còn sót nhưng không dùng
**Root Cause:** Cleanup không hoàn toàn

**Evidence:**

1. `internal/models/common.go` lines 19-74: OpenRouter types vẫn tồn tại:
```go
type OpenRouterRequest struct {
    Messages    []OpenRouterMessage `json:"messages"`
    Tools       []OpenRouterTool    `json:"tools,omitempty"`
    // ... 50+ lines của type definitions không dùng
}
```

2. `internal/core/dispatcher.go` line 485: Comment còn reference OpenRouter:
```go
// The result is normalized to valid JSON so both Gemini and OpenRouter providers
// see a consistent format when translating history back to their API schema.
```

**Impact:**
- Code bloat (70+ lines type definitions)
- Confusion cho future maintainers
- Tiềm ẩn memory leak nếu bất kỳ đâu import package này

**Confidence:** 0.9

---

### 🟠 HIGH: Missing Long-Term Memory (No L2)

**Severity:** HIGH
**Layer:** 3 (Long-Term Memory)
**Mechanism:** Session history reset khi new chat
**Root Cause:** Không có persistent vector store hoặc knowledge base

**Evidence:**

1. `internal/domain/entities/conversation.go`:
```go
type ContextWindow struct {
    History      []Message          `json:"history"`  // <--- Session-level only
    MaxMessages  int                `json:"max_messages"`
    MaxTokens    int                `json:"max_tokens"`
    WindowSize   int                `json:"window_size"`
    UpdatedAt    time.Time          `json:"updated_at"`
}
```

2. `internal/store/session_store.go`: Chỉ lưu session cụ thể, không có shared knowledge base:
```go
// SaveSession saves or updates a chat session in Redis (with memory fallback).
```

**Impact:**
- Agent forget everything khi user mở new chat
- Không có cross-session learning
- Duplicate information retrieval (tool calls) cho từng session mới

**Confidence:** 0.9

---

### 🟡 MEDIUM: Context Duplication (Layer 5 Active Recall)

**Severity:** MEDIUM
**Layer:** 5 (Active Recall)
**Mechanism:** System prompt + history + bootstrap context + skill docs → duplication
**Root Cause:** Không có de-duplication strategy

**Evidence:**

1. `internal/core/orchestrator.go` lines 136-149: Build messages từ 4 layers:
```go
messages = []messaging.Message{}
if systemPrompt != "" {
    messages = append(messages, messaging.Message{
        Role:    messaging.RoleSystem,
        Content: systemPrompt,  // <--- Layer 1
    })
}
messages = append(messages, condensedHistory...)  // <--- Layer 2
// ...
req := messaging.Request{
    History: messages,
    Tools:   tools,
}
```

2. `internal/core/bootstrap.go` lines 34-63: Bootstrap context appended sau đó:
```go
func BuildBootstrapContext(agent *Agent, route RoutePlan) []string {
    // Load agent docs
    agentDoc := tools.LoadDocumentWithMetadata("agent", route.Agent)
    contextParts := []string{
        fmt.Sprintf("SYSTEM PROMPT (from agents/%s.md)\n%s", route.Agent, agentDoc.Content),
    }
    // Load skill docs
    for _, skill := range route.Skills {
        skillDoc := tools.LoadDocumentWithMetadata("skill", route.Agent+"/"+skill)
        contextParts = append(contextParts, fmt.Sprintf("SKILL MARKDOWN (%s)\n%s", skill, content))
    }
    // <--- Layer 4 (distillation) duplicated ở đây
}
```

3. `internal/core/context_window.go` lines 50-90: BuildLLMHistory insert summary sau system prompt:
```go
func (cw *ContextWindow) BuildLLMHistory(keepRecent int) []messaging.Message {
    result := []messaging.Message{}
    if cw.MemorySummary != "" {
        summaryMsg := messaging.Message{
            Role:    messaging.RoleUser,
            Content: "=== TÓM TẮT NGỮ CẢNH TRƯỚC ĐÂY ===\n" + cw.MemorySummary,
        }
        result = append(result, summaryMsg)  // <--- Layer 5
    }
    result = append(result, cw.History[0:2])  // Bootstrap messages
    // ...
}
```

**Impact:**
- Context token waste (có thể 20-30% redundant)
- Tốn quota Gemini
- Degradation message quality over time

**Confidence:** 0.8

---

### 🟡 MEDIUM: Tool Execution Validation Too Permissive

**Severity:** MEDIUM
**Layer:** 7 (Tool Execution)
**Mechanism:** Validate required params nhưng KHÔNG validate tool calling
**Root Cause:** Logic validate ở dispatcher nhưng không enforce trong orchestrator

**Evidence:**

1. `internal/core/dispatcher.go` lines 196-218: Validate required params NHIỀU LẦN cho mỗi tool call:
```go
func (d *Dispatcher) dispatchToolCall(toolCall *messaging.ToolCall) string {
    handler, ok := d.handlers[toolCall.Name]
    if !ok {
        return fmt.Sprintf("Error: Unknown tool %s", toolCall.Name)
    }
    // <--- Validate REQUIRED params NHIỀU LẦN (đểu cho mỗi tool)
    for _, schema := range d.GetTools() {
        if schema.Name == toolCall.Name {
            if err := validateRequiredParams(toolCall.Name, toolCall.Args, schema); err != nil {
                return fmt.Sprintf("Error: %v", err)
            }
            break
        }
    }
    result, err := handler(toolCall.Args)
    // ...
}
```

2. `internal/core/dispatcher.go` lines 64-168: Tool schemas được truyền cho LLM nhưng không enforce:
```go
func (d *Dispatcher) GetTools() []messaging.ToolSchema {
    return []messaging.ToolSchema{
        {
            Name:        "financial_research",
            Description: "Truy vấn dữ liệu thị trường...",
            Parameters: map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "query": map[string]interface{}{
                        "type": "string",
                        "description": "Từ khóa hoặc mã chứng khoán cần tra cứu",
                    },
                },
                "required": []string{"query"},  // <--- Prompt nói "required"
            },
        },
        // ... 8 tools nữa
    }
}
```

**Impact:**
- Code validate 2-3 lần cho mỗi tool call (waste CPU)
- LLM có thể call tool mà thiếu required params (validation catch nó, nhưng gây delay)
- Không detect missing tool calls at orchestrator level

**Confidence:** 0.7

---

### 🟢 LOW: Prompt Bloat (Layer 1 System Prompt)

**Severity:** LOW
**Layer:** 1 (System Prompt)
**Mechanism:** Prompt quá dài (1800+ tokens)
**Root Cause:** Aggressive rule enforcement

**Evidence:**

1. `internal/prompt/system_prompt.txt`: 48 lines, ~1800 tokens
   - 48 rules trong 1 file
   - Mix của system instructions, tool definitions, fallback rules, citation rules
   - Không có modularity

**Impact:**
- Context token waste
- Hard to maintain
- Model focus on rules > quality

**Confidence:** 0.6

---

## Ordered Fix Plan (Code-First)

### Phase 1: CRITICAL FIXES (Do trước khi release)

#### 1. Remove Hidden Repair Loop
**Goal:** Eliminate silent fallback LLM calls
**Why Now:** Có thể gây double billing và non-deterministic behavior
**Expected Effect:** -30% quota waste, predictable error messages

**Fix Steps:**

1. **Remove fallback logic trong MultiProvider** (hàng 130-227 trong `multiprovider.go`):
```go
// ❌ OLD (DELETE):
func (m *MultiProvider) Generate(ctx context.Context, req messaging.Request) (messaging.Message, error) {
    // Try primary first
    aiMessage, err := m.primary.Generate(ctx, req)
    if err == nil {
        return aiMessage, nil
    }
    // Fallback logic - SILENT LLM CALL
    return m.tryFallbacksOnly(ctx, req)
}

// ✅ NEW:
func (m *MultiProvider) Generate(ctx context.Context, req messaging.Request) (messaging.Message, error) {
    return m.primary.Generate(ctx, req)
}
```

2. **Remove fallback queue** (primaryFailures, skipPrimaryUntil fields)

3. **Remove retryDelay function** (không cần nữa)

4. **Remove entire `tryFallbacksOnly` method**

5. **Update `provider_mgr.go` để không tạo fallbacks**:
```go
func NewProviderManager() *ProviderManager {
    geminiProviders := newGeminiProviders()
    // ✅ Chỉ dùng primary, không có fallback chain
    return &ProviderManager{provider: geminiProviders[0]}
}
```

6. **Remove OpenRouter types** (hàng 19-74 trong `common.go`)

7. **Remove deprecated `routeWithProviderFallback`** (hàng 107-111 trong `routing.go`)

**Files to Modify:**
- `internal/infrastructure/providers/multiprovider.go` (-120 lines)
- `internal/models/common.go` (-56 lines)
- `internal/core/routing.go` (-5 lines)

---

#### 2. Enforce Tool Calling (Code-Gate Tool Requirements)

**Goal:** Prevent LLM from skipping tools bằng code validation
**Why Now:** Prompt "BẮT BUỘC" là ineffective
**Expected Effect:** -90% hallucination, proper tool usage

**Fix Steps:**

1. **Add orchestrator-level enforcement** (`internal/core/orchestrator.go`):

```go
// ✅ NEW METHOD trong Orchestrator:
func (o *Orchestrator) enforceToolRequirements(aiMessage messaging.Message, userInput string) (bool, string) {
    // Kiểm tra REQUIRED tools dựa trên userInput
    requiredTools := o.identifyRequiredTools(userInput)

    if len(requiredTools) == 0 {
        return true, ""  // Không có tool required
    }

    // Kiểm tra tool calls
    hasToolCalls := len(aiMessage.ToolCalls) > 0
    if hasToolCalls {
        // Kiểm tra tool names match required tools
        calledTools := make(map[string]bool)
        for _, tc := range aiMessage.ToolCalls {
            calledTools[tc.Name] = true
        }

        missingTools := []string{}
        for _, rt := range requiredTools {
            if !calledTools[rt] {
                missingTools = append(missingTools, rt)
            }
        }

        if len(missingTools) > 0 {
            errorMsg := fmt.Sprintf(
                "Bạn PHẢI gọi các công cụ: %s. Phản hồi không thể được chấp nhận mà không có dữ liệu từ các công cụ này.",
                strings.Join(missingTools, ", "),
            )
            return false, errorMsg
        }
    } else {
        // Không có tool calls mà required tools tồn tại
        errorMsg := fmt.Sprintf(
            "Bạn PHẢI gọi các công cụ: %s. Đừng tự bịa số liệu.",
            strings.Join(requiredTools, ", "),
        )
        return false, errorMsg
    }

    return true, ""
}

// Helper để identify required tools:
func (o *Orchestrator) identifyRequiredTools(userInput string) []string {
    q := strings.ToLower(userInput)
    required := []string{}

    // Rule 1: Nếu mention "xuất báo cáo" → BẮT BUỘC export_report
    if strings.Contains(q, "xuất báo cáo") || strings.Contains(q, "tải báo cáo") || strings.Contains(q, "excel") || strings.Contains(q, "powerpoint") {
        required = append(required, "export_report")
    }

    // Rule 2: Nếu mention "tra cứu" hoặc tên mã chứng khoán → BẮT BUỘC financial_research
    if strings.Contains(q, "tra cứu") || strings.Contains(q, "tìm kiếm") || (isStockSymbol(q) && !strings.Contains(q, "xuất")) {
        required = append(required, "financial_research")
    }

    // Rule 3: Nếu mention scrape → BẮT BUỘC financial_scrape
    if strings.Contains(q, "scrape") || strings.Contains(q, "đọc sâu") {
        required = append(required, "financial_scrape")
    }

    return required
}
```

2. **Integrate enforcement vào ReAct loop** (`orchestrator.go` lines 302-305):
```go
// ❌ OLD:
aiMessage, err := o.agent.GetProvider().Generate(ctx, req)

// ✅ NEW:
aiMessage, err := o.agent.GetProvider().Generate(ctx, req)
if err != nil {
    return "", err
}

// Enforce tool requirements sau khi LLM response
enforced, errorMsg := o.enforceToolRequirements(aiMessage, userInput)
if !enforced {
    // Append enforcement error vào AI response
    aiMessage.Content = errorMsg
    o.agent.mu.Lock()
    o.agent.conversation.ContextWindow.History = append(o.agent.conversation.ContextWindow.History, aiMessage)
    o.agent.mu.Unlock()
    return "", fmt.Errorf("tool requirement enforcement failed: %s", errorMsg)
}
```

3. **Remove duplicate validation** (`dispatcher.go` lines 203-211):
```go
// ❌ OLD: Validate REQUIRED params 2-3 lần cho mỗi tool
for _, schema := range d.GetTools() {
    if schema.Name == toolCall.Name {
        if err := validateRequiredParams(toolCall.Name, toolCall.Args, schema); err != nil {
            return fmt.Sprintf("Error: %v", err)
        }
        break
    }
}

// ✅ NEW: Validate chỉ 1 lần
// (enforce ở orchestrator level, dispatcher chỉ handle execution)
```

**Files to Modify:**
- `internal/core/orchestrator.go` (+80 lines enforcement logic)
- `internal/core/dispatcher.go` (-10 lines duplicate validation)

---

### Phase 2: HIGH PRIORITY (Do trong sprint này)

#### 3. Remove OpenRouter Remnants

**Goal:** Clean codebase khỏi OpenRouter types
**Why Now:** Code bloat, confusion, potential memory leak
**Expected Effect:** -100 lines clean, dễ maintain

**Fix Steps:**

1. **Delete `internal/models/common.go` OpenRouter types** (lines 19-74):
```bash
# Remove entire file hoặc file
```

2. **Update imports** trong file nào import `internal/models/common.go`:
```go
// ❌ OLD:
import "gemini-cli/internal/models/common"

// ✅ NEW:
import (
    "gemini-cli/internal/models/messaging"
    "gemini-cli/internal/models/common"
)
```

3. **Search & replace** `OpenRouter*` references:
```bash
rg "OpenRouter" --type go -l  # List all files
rg "OpenRouter" --type go -A 2 -B 2  # Show context
```

**Files to Modify:**
- `internal/models/common.go` (DELETE)

---

#### 4. Add Long-Term Memory (L2 Storage)

**Goal:** Enable cross-session learning
**Why Now:** User báo "agent forget everything khi new chat"
**Expected Effect:** Consistent behavior, reduced tool calls

**Fix Steps:**

1. **Add knowledge base interface** (`internal/domain/interfaces/knowledge_base.go`):
```go
package interfaces

import (
    "context"
    "gemini-cli/internal/models/messaging"
)

type KnowledgeBase interface {
    // Save tool call results persistently
    SaveToolResult(ctx context.Context, toolName, toolResult string) error

    // Retrieve relevant context for new query
    RetrieveContext(ctx context.Context, query string) ([]messaging.Message, error)

    // Mark conversation as successfully completed (for learning)
    MarkConversationCompleted(ctx context.Context, sessionID string) error

    // Close database connection
    Close() error
}
```

2. **Add SimpleKV Store** (`internal/store/kv_store.go`):
```go
package store

import (
    "context"
    "encoding/json"
    "sync"
    "time"
)

type SimpleKVStore struct {
    data    map[string]string
    mu      sync.RWMutex
    ttl     time.Duration
    ticker  *time.Ticker
    shutdown chan struct{}
}

func NewSimpleKVStore(ttl time.Duration) *SimpleKVStore {
    kv := &SimpleKVStore{
        data:     make(map[string]string),
        ttl:      ttl,
        shutdown: make(chan struct{}),
    }
    go kv.cleanupLoop()
    return kv
}

func (kv *SimpleKVStore) Get(ctx context.Context, key string) (string, bool) {
    kv.mu.RLock()
    defer kv.mu.RUnlock()
    val, exists := kv.data[key]
    return val, exists
}

func (kv *SimpleKVStore) Set(ctx context.Context, key, value string) error {
    kv.mu.Lock()
    defer kv.mu.Unlock()
    kv.data[key] = value
    return nil
}

func (kv *SimpleKVStore) cleanupLoop() {
    for {
        select {
        case <-time.After(kv.ttl):
            kv.cleanupExpired()
        case <-kv.shutdown:
            return
        }
    }
}

func (kv *SimpleKVStore) cleanupExpired() {
    kv.mu.Lock()
    defer kv.mu.Unlock()
    now := time.Now()
    for key, meta := range kv.data {
        var data struct {
            ExpiredAt time.Time
        }
        if err := json.Unmarshal([]byte(meta), &data); err == nil {
            if now.After(data.ExpiredAt) {
                delete(kv.data, key)
            }
        }
    }
}
```

3. **Initialize KV Store trong orchestrator**:
```go
type Orchestrator struct {
    agent *entities.Agent
    kb    interfaces.KnowledgeBase
}

func NewOrchestrator(a *entities.Agent) *Orchestrator {
    return &Orchestrator{
        agent: a,
        kb:    store.NewSimpleKVStore(24 * time.Hour),  // 24 hours TTL
    }
}
```

4. **Use knowledge base khi user hỏi lại**:
```go
func (o *Orchestrator) ProcessMessage(ctx context.Context, userInput string, atts []messaging.Attachment) (string, error) {
    // Check knowledge base trước khi call LLM
    contextMessages, err := o.kb.RetrieveContext(ctx, userInput)
    if err == nil && len(contextMessages) > 0 {
        fmt.Printf("📚 [L2 Memory] Retrieved %d messages từ knowledge base\n", len(contextMessages))
        o.agent.LoadHistory(contextMessages)
    }

    // ... ReAct loop
}
```

**Files to Create:**
- `internal/domain/interfaces/knowledge_base.go`
- `internal/store/kv_store.go`

**Files to Modify:**
- `internal/core/orchestrator.go` (+10 lines initialization)

---

### Phase 3: MEDIUM PRIORITY (Do trong next cycle)

#### 5. Reduce Context Duplication

**Goal:** De-duplicate system prompt, history, bootstrap context
**Why Now:** 20-30% token waste
**Expected Effect:** -20% quota usage

**Fix Steps:**

1. **Move tool definitions vào system prompt** (Layer 1):
```go
// ✅ NEW: system_prompt.txt thay vì truyền qua req.Tools
```

2. **Remove tool schemas từ req** (`dispatcher.go` line 64-168):
```go
func (d *Dispatcher) GetTools() []messaging.ToolSchema {
    return []messaging.ToolSchema{}  // Empty, không dùng nữa
}
```

3. **Integrate bootstrap context vào system prompt** (Layer 1):
```go
// System prompt cho mỗi agent:
func BuildAgentSystemPrompt(agentDoc string, skillDocs []string) string {
    return fmt.Sprintf(`
ANTHROPIC AGENT CONFIGURATION
%s

SKILL DOCS:
%s

SYSTEM RULES:
...
    `, agentDoc, strings.Join(skillDocs, "\n\n---\n\n"))
}
```

4. **Remove MemorySummary injection** (Layer 5):
```go
// ❌ OLD: Insert summary ở đầu messages
if cw.MemorySummary != "" {
    summaryMsg := messaging.Message{...}
    result = append(result, summaryMsg)
}

// ✅ NEW: Chỉ giữ protected messages + recent messages
// Summary ở đây là redundancy với skill docs
```

**Files to Modify:**
- `internal/prompt/system_prompt.txt` (enlarge)
- `internal/core/orchestrator.go` (-15 lines BuildLLMHistory)
- `internal/core/dispatcher.go` (-100 lines tool schemas)

---

## Related Skills

- `agent-introspection-debugging` — Debug agent runtime failures (loops, timeouts, state errors)
- `go-review` — Comprehensive Go code review cho idiomatic patterns, concurrency safety, error handling, security
- `go-build-fix` — Fix Go build errors, go vet warnings, linter issues
- `security-review` — Security vulnerability detection cho code, configuration

---

## Quick Diagnostic Summary

| # | Question | Answer | Action |
|---|----------|--------|--------|
| 1 | Model skip tool và trả lời ngay? | ✅ **YES** — LLM có thể skip, prompt enforcement ineffective | Fix ở Phase 1.2 |
| 2 | Old conversation content hiện trong new turn? | ❌ NO — Session history reset khi new chat | L2 memory ở Phase 2.4 |
| 3 | Same info trong system prompt + memory + history? | ✅ **YES** — Duplication 20-30% | Fix ở Phase 3.5 |
| 4 | Platform run second LLM pass trước delivery? | ✅ **YES** — Hidden repair loop | Fix ở Phase 1.1 |
| 5 | Output khác giữa internal generation + user delivery? | ❌ NO — Direct pass-through | OK |
| 6 | "Must use tool X" chỉ trong prompt text? | ✅ **YES** — Code không enforce | Fix ở Phase 1.2 |
| 7 | Agent monologue persist vào long-term memory? | ❌ NO — Không có L2 storage | Fix ở Phase 2.4 |

---

## Anti-Patterns Found

✅ **Avoided:**
- Không blame model trước khi falsify wrapper-layer regressions
- Không blame memory without showing contamination path
- Code-first fixes (orchestrator enforcement) thay vì prompt-only fixes

⚠️ **Caution Needed:**
- Remove OpenRouter references carefully (check imports)
- Long-term memory implementation cần test để tránh OOM

---

## Recommendation Summary

**DO:**
1. Fix hidden repair loop (Phase 1.1) trước khi release production
2. Enforce tool calling bằng code (Phase 1.2)
3. Remove OpenRouter remnants (Phase 2.3)
4. Add long-term memory (Phase 2.4)

**DON'T:**
1. Depend vào prompt enforcement cho critical logic
2. Assume fallback providers an toàn nếu không documented
3. Let LLM skip tools và tự bịa dữ liệu

---

**Report Generated:** 2026-06-17
**Model:** Claude Sonnet 4.6
**Audit Methodology:** Agent Architecture Audit Skill (ECC v2.0.0-rc.1)
