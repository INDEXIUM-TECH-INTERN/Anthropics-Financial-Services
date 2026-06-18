# Architecture Fix Plan - Implementation Roadmap

**Based On:** Architecture Audit Report (2026-06-16)
**Priority Order:** Code-first, not prompt-first
**Estimated Effort:** 4-6 sprints (16-24 days)

---

## Phase 1: Tool Discipline Enforcement (Priority 1)

**Goal:** Code-gate mandatory tool usage to prevent data fabrication.

### Sub-tasks

#### 1.1: Implement Tool Usage Validation

**Location:** `internal/core/dispatcher.go`

**What to Do:**
1. Add `ValidateToolUsage()` method that checks if model called required tools
2. Implement query analysis to detect when tools are mandatory:
   - Time queries → `time_parser` tool required
   - Financial data queries → `financial_research` or `tavily_search` required
   - Data extraction queries → `financial_scrape` required
3. Return error if required tools not called
4. Implement retry mechanism for validation failures

**Code Structure:**
```go
type Dispatcher struct {
    agent *Agent
    cache *cache.LRUCache
    handlers map[string]toolHandler
    // Add these fields
    timeParserRequired bool
    researchRequired bool
}

func (d *Dispatcher) AnalyzeQueryForToolRequirements(history []messaging.Message) error {
    // Analyze last user message for intent
    // Set required tools flags
}

func (d *Dispatcher) ValidateToolUsage(req *messaging.Request, response *messaging.Message) error {
    // Check if required tools were called
    // Return error if not
}
```

**Files to Create:**
- `internal/core/tool_validator.go` (new file)

**Files to Modify:**
- `internal/core/dispatcher.go` (add validation method)
- `internal/core/agent.go` (integrate validation into ReAct loop)
- `internal/core/orchestrator.go` (wrap validation in loop)

**Test Files:**
- `internal/core/tool_validator_test.go` (new)
- Update `internal/core/dispatcher_test.go`

**Estimated Effort:** 3-4 days

---

#### 1.2: Add Unit Tests

**What to Do:**
1. Test validation for time queries without time parser
2. Test validation for financial data queries without research tools
3. Test validation for scrape queries without scrape tool
4. Test retry mechanism on validation failure
5. Test edge cases (empty messages, no tools defined)

**Test Coverage Targets:**
- Validation logic: 100%
- Retry scenarios: 80%
- Edge cases: 60%

**Test Structure:**
```go
func TestValidateToolUsage_TimeQueryWithoutTimeParser(t *testing.T) {
    // Setup mock agent with time query
    // Call ValidateToolUsage
    // Expect error
}

func TestValidateToolUsage_RetryOnValidationFailure(t *testing.T) {
    // Simulate validation failure
    // Verify retry happens
    // Verify final response includes tool calls
}
```

**Estimated Effort:** 2 days

---

#### 1.3: Integration Testing

**What to Do:**
1. Test full ReAct loop with validation enabled
2. Verify tool calls appear when validation fails
3. Test timeout behavior on validation loop
4. Test with concurrent sessions

**Estimated Effort:** 1 day

---

### Phase 1 Deliverables

**Acceptance Criteria:**
- [ ] Validation catches all missing mandatory tools
- [ ] Retry mechanism works correctly
- [ ] Tool calls appear after retry
- [ ] No regression in existing functionality
- [ ] 80%+ test coverage for validation logic
- [ ] Documentation updated (README, architecture docs)

**Deliverables:**
- `internal/core/tool_validator.go`
- `internal/core/tool_validator_test.go`
- Updated `internal/core/dispatcher.go` (with validation)
- Updated `internal/core/agent.go` (validation integration)
- Updated `internal/core/orchestrator.go` (validation loop)
- `docs/TOOL_VALIDATION.md` (documentation)

**Estimated Total:** 6-7 days

---

## Phase 2: Prompt Context Consolidation (Priority 2)

**Goal:** Eliminate redundant content across prompt layers.

### Sub-tasks

#### 2.1: Create Single Source of Truth for Time Rules

**Location:** `internal/prompt/time_rules.txt`

**What to Do:**
1. Extract time rules from `enhanced_time_prompt.txt` (lines 1-55)
2. Consolidate into clean, documented format
3. Use bullet points for readability
4. Include examples and edge cases
5. Add "Last Updated" date and version

**Content Structure:**
```
# Time Calculation Rules - Version 1.0

## System Time
- Current year: {{CURRENT_YEAR}}
- Years: {{YEAR_MINUS_1}}, {{YEAR_MINUS_2}}, etc.
- Weekday: {{SYSTEM_WEEKDAY}}

## Time Expressions
- "3 years": {{YEAR_MINUS_2}}, {{YEAR_MINUS_1}}, {{CURRENT_YEAR}}
- "5 years": {{YEAR_MINUS_4}} to {{CURRENT_YEAR}}
- "Next quarter": 3 months from now
- "Last year": {{YEAR_MINUS_1}}

## Data Currency Rules
- Before 2020: Consider outdated
- 2024: Warning required (unless current year is 2025+)
- Current year: Data still being populated

## Special Cases
- Trading hours: 9:00-15:00 Mon-Fri
- Future dates: Always relative to now
- Past dates: Don't trust beyond 6 months
```

**Files to Create:**
- `internal/prompt/time_rules.txt`

**Estimated Effort:** 0.5 day

---

#### 2.2: Update enhanced_time_prompt.txt

**What to Do:**
1. Replace manual rule duplication with Go template include
2. Keep only high-level rules and exceptions
3. Add reference to `time_rules.txt` for detailed documentation
4. Update version/last-updated fields

**Go Template Example:**
```go
// in orchestrator.go
type TimePromptData struct {
    CurrentYear    string
    YearMinus1     string
    YearMinus2     string
    YearMinus4     string
    CurrentTime    string
    Weekday        string
}

func (o *Orchestrator) loadEnhancedTimePrompt(data TimePromptData) string {
    template := `{{base_system_prompt}}

=== TIME CALCULATION RULES ===

System Time:
- Current Year: {{.CurrentYear}}
- Reference Years: {{.YearMinus1}}, {{.YearMinus2}}, {{.YearMinus4}}

Time Expressions:
{{time_rules_yaml}}

For detailed rules, see internal/prompt/time_rules.txt

=== TIME SAFETY RULES ===

MUST NOT:
- Use data before 2020
- Use year 2024 when current year is 2025+

MUST:
- Use GetCurrentTimeInfo()
- Call time_parser tool when needed
- Check trading hours

`
    return template
}
```

**Files to Modify:**
- `internal/core/orchestrator.go` (add template loading)
- `internal/prompt/enhanced_time_prompt.txt` (simplify to reference)

**Estimated Effort:** 1 day

---

#### 2.3: Add CI Check for Prompt Consistency

**What to Do:**
1. Create GitHub Action workflow `.github/workflows/prompt-check.yml`
2. Add check that detects duplicate content between prompts
3. Add check that ensures `time_rules.txt` is referenced in prompts
4. Fail CI if:
   - Duplicate text found in prompts
   - `time_rules.txt` not referenced when it should be
   - Broken template syntax

**Workflow Structure:**
```yaml
name: Prompt Consistency Check

on: [pull_request]

jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Check for duplicate content
        run: |
          # Find duplicate text across prompt files
          # Use fuzzy string matching
          node scripts/check-prompt-duplicates.mjs

      - name: Check time_rules.txt reference
        run: |
          # Ensure enhanced_time_prompt.txt references time_rules.txt
          grep -q "time_rules.txt" internal/prompt/enhanced_time_prompt.txt
```

**Files to Create:**
- `.github/workflows/prompt-check.yml`
- `scripts/check-prompt-duplicates.mjs`

**Estimated Effort:** 1 day

---

#### 2.4: Documentation

**What to Do:**
1. Create `docs/PROMPT_ARCHITECTURE.md`
2. Document single source of truth principle
3. Explain prompt template system
4. Add guide for adding new rules
5. Update README with prompt structure

**Documentation Structure:**
```markdown
# Prompt Architecture

## Principles
1. Single Source of Truth: Each rule defined in one place
2. Template-Based: Use Go templates for dynamic values
3. Documentation: Every rule documented
4. CI Checks: Automated validation

## Prompt Files
- `system_prompt.txt`: Core agent behavior (100% enforced)
- `enhanced_time_prompt.txt`: Time-specific rules (uses time_rules.txt)
- `time_rules.txt`: Detailed time calculation rules (single source)
- `...`: Other specialized prompts

## Adding New Rules
1. Define rule in appropriate file (or new file)
2. Update CI check if needed
3. Document in PROMPT_ARCHITECTURE.md
4. Update CHANGELOG
```

**Files to Create:**
- `docs/PROMPT_ARCHITECTURE.md`
- Update `README.md`

**Estimated Effort:** 1 day

---

### Phase 2 Deliverables

**Acceptance Criteria:**
- [ ] All time rules in single `time_rules.txt` file
- [ ] No duplicate content between prompts
- [ ] CI check fails on duplicate content
- [ ] Documentation complete and clear
- [ ] All prompts use Go templates where appropriate

**Deliverables:**
- `internal/prompt/time_rules.txt`
- `internal/prompt/enhanced_time_prompt.txt` (updated)
- `internal/core/orchestrator.go` (template support)
- `.github/workflows/prompt-check.yml`
- `scripts/check-prompt-duplicates.mjs`
- `docs/PROMPT_ARCHITECTURE.md`

**Estimated Total:** 3.5 days

---

## Phase 3: Session-Aware Cache Eviction (Priority 3)

**Goal:** Prevent cache starvation for new sessions.

### Sub-tasks

#### 3.1: Enhance LRU Cache for Session Awareness

**Location:** `internal/cache/lru.go`

**What to Do:**
1. Add `sessionID` parameter to `Get()`, `Set()`, `Delete()`
2. Update internal key structure: `sessionID + ":" + cacheKey`
3. Implement session-aware eviction policy:
   - Each session has own LRU list
   - Global limit of 200 entries total
   - Evict from least recently used session
4. Add metrics: `cache_hits_per_session`, `cache_evictions_per_session`

**Code Structure:**
```go
type SessionLRU struct {
    sessions sync.Map // map[string]*sessionCache
    maxEntries int
}

type sessionCache struct {
    lru *LRUCache
    accessOrder []string // for eviction
}

func (sc *SessionLRU) Get(sessionID, key string) (string, bool) {
    fullKey := sessionID + ":" + key
    // Use LRU from session
}

func (sc *SessionLRU) Set(sessionID, key, value string) {
    fullKey := sessionID + ":" + key
    // Set with LRU update
}

func (sc *SessionLRU) EvictLeastUsed() (string, string) {
    // Find session with oldest access
    // Evict from that session's LRU
}
```

**Files to Modify:**
- `internal/cache/lru.go` (add session support)

**Estimated Effort:** 2 days

---

#### 3.2: Update Dispatcher Cache Usage

**Location:** `internal/core/dispatcher.go`

**What to Do:**
1. Add `sessionID` field to `Dispatcher`
2. Pass `sessionID` to all cache calls
3. Create new cache per session (or per-agent-session combination)
4. Add cache metrics logging

**Code Structure:**
```go
type Dispatcher struct {
    agent     *Agent
    cache     *SessionLRU
    sessionID string // Add this
    // ... existing fields
}

func NewDispatcher(a *Agent, sessionID string) *Dispatcher {
    d := &Dispatcher{
        agent:    a,
        cache:    NewSessionLRU(maxCacheEntries),
        sessionID: sessionID,
    }
    d.registerHandlers()
    return d
}
```

**Files to Modify:**
- `internal/core/dispatcher.go`
- `internal/core/agent.go` (pass sessionID)
- `internal/api/handlers.go` (pass sessionID to dispatcher)

**Estimated Effort:** 1 day

---

#### 3.3: Add Concurrent Session Tests

**What to Do:**
1. Test cache isolation between sessions
2. Test eviction when multiple sessions compete
3. Test cache hit rate fairness
4. Test cache cleanup on session close

**Test Structure:**
```go
func TestSessionCacheIsolation(t *testing.T) {
    cache := NewSessionLRU(200)

    // Session A operations
    cache.Set("sessionA", "key1", "value1")
    value, ok := cache.Get("sessionA", "key1")
    assert.True(t, ok)
    assert.Equal(t, "value1", value)

    // Session B should not see Session A's cache
    value, ok = cache.Get("sessionB", "key1")
    assert.False(t, ok)
}

func TestSessionEvictionFairness(t *testing.T) {
    // Create 5 sessions with 100 operations each
    // Verify all sessions get fair cache space
}
```

**Files to Modify:**
- `internal/core/dispatcher_test.go` (add tests)
- `internal/cache/lru_test.go` (add tests)

**Estimated Effort:** 1 day

---

#### 3.4: Monitoring and Metrics

**What to Do:**
1. Add metrics to Prometheus/Grafana
2. Track cache hit rate per session
3. Track eviction frequency
4. Set up alert if eviction rate > 50%

**Files to Create:**
- `internal/metrics/cache_metrics.go`

**Estimated Effort:** 1 day

---

### Phase 3 Deliverables

**Acceptance Criteria:**
- [ ] Each session has isolated cache
- [ ] Cache eviction is fair across sessions
- [ ] Cache hit rate monitored per session
- [ ] No regression in cache performance
- [ ] Tests verify isolation and fairness

**Deliverables:**
- `internal/cache/lru.go` (session-aware)
- `internal/core/dispatcher.go` (updated)
- `internal/metrics/cache_metrics.go` (new)
- `internal/core/dispatcher_test.go` (updated)
- `internal/cache/lru_test.go` (updated)

**Estimated Total:** 5 days

---

## Phase 4: Enforce Time Knowledge Validation (Priority 4)

**Goal:** Make time warnings enforceable, not cosmetic.

### Sub-tasks

#### 4.1: Implement Retry on Time Validation

**Location:** `internal/core/orchestrator.go`

**What to Do:**
1. Add time validation check after ReAct loop
2. If time issues detected, retry loop instead of just warning
3. Limit retry attempts to 2-3 iterations
4. Add detailed error message to model if retry fails

**Code Structure:**
```go
func (o *Orchestrator) streamFinalResponse(ctx context.Context, onChunk func(string, bool)) error {
    // ... existing loop ...

    for i := 0; i < maxIterations; i++ {
        // ... tool calling ...

        // NEW: Check time validation
        timeIssues := o.agent.CheckTimeKnowledgeIssues(finalText)

        if timeIssues != "" {
            if o.timeValidationRetries < 2 {
                // Retry without incrementing iteration count
                // Add timeIssues to context
                continue
            }
            // Final warning if max retries exceeded
            finalText += "\n\n" + timeIssues
        }

        // ... streaming ...
    }
}
```

**Files to Modify:**
- `internal/core/orchestrator.go`
- `internal/core/agent.go` (add retry counter)

**Estimated Effort:** 1 day

---

#### 4.2: Add Time Validation Tests

**What to Do:**
1. Test retry on outdated time usage
2. Test retry on wrong year usage
3. Test retry limit (max 2 retries)
4. Test final warning after max retries

**Test Structure:**
```go
func TestTimeValidation_RetryOnWrongYear(t *testing.T) {
    // Setup agent with outdated year in response
    // Run streamFinalResponse
    // Verify retry happened
    // Verify final warning present
}

func TestTimeValidation_MaxRetries(t *testing.T) {
    // Setup agent that always produces wrong year
    // Run streamFinalResponse
    // Verify max retries reached
    // Verify final warning
}
```

**Files to Modify:**
- `internal/core/orchestrator_test.go`

**Estimated Effort:** 1 day

---

#### 4.3: Documentation Update

**What to Do:**
1. Update architecture docs to reflect time validation enforcement
2. Add troubleshooting guide for time-related failures
3. Document retry logic

**Files to Modify:**
- `docs/ARCHITECTURE.md`
- `docs/TIME_VALIDATION.md` (new)

**Estimated Effort:** 0.5 day

---

### Phase 4 Deliverables

**Acceptance Criteria:**
- [ ] Time validation triggers retry (not just warning)
- [ ] Max 2 retries enforced
- [ ] Final warning when max retries reached
- [ ] Tests verify retry behavior
- [ ] Documentation updated

**Deliverables:**
- `internal/core/orchestrator.go` (updated)
- `internal/core/agent.go` (updated)
- `internal/core/orchestrator_test.go` (updated)
- `docs/TIME_VALIDATION.md` (new)

**Estimated Total:** 2.5 days

---

## Summary of Effort

| Phase | Focus | Days | Priority |
|-------|-------|------|----------|
| 1 | Tool Discipline Enforcement | 6-7 days | **CRITICAL** |
| 2 | Prompt Context Consolidation | 3.5 days | HIGH |
| 3 | Session-Aware Cache | 5 days | MEDIUM |
| 4 | Time Validation Enforcement | 2.5 days | MEDIUM |
| **TOTAL** | | **17-18 days** | |

---

## Sprint Planning (4 Sprints × 4-5 days)

### Sprint 1: Tool Discipline Enforcement
- Days 1-3: Validation implementation
- Days 4-5: Testing

### Sprint 2: Prompt Consolidation
- Days 1-2: Create time_rules.txt
- Days 3-4: Update prompts and templates
- Day 5: CI check and documentation

### Sprint 3: Session-Aware Cache
- Days 1-2: Enhance LRU cache
- Days 3-4: Update dispatcher and tests
- Day 5: Metrics and monitoring

### Sprint 4: Time Validation & Documentation
- Days 1-2: Implement retry logic
- Day 3: Testing
- Days 4-5: Final documentation and review

---

## Risk Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| Tool validation blocks valid responses | HIGH | Add extensive logging; allow configuration flag to disable validation for testing |
| Template errors break prompts | MEDIUM | CI checks for template syntax; rollback plan with version tags |
| Session cache complexity bugs | MEDIUM | Comprehensive tests; A/B testing before production deployment |
| Time validation too aggressive | LOW | Limit retries; make validation opt-in by default |

---

## Success Criteria

### Before Production
1. ✅ Tool validation passes 95%+ of test cases
2. ✅ Prompt consistency CI check passes always
3. ✅ Session cache tests verify isolation and fairness
4. ✅ Time validation tests cover all edge cases
5. ✅ Documentation complete and reviewed
6. ✅ No regression in existing functionality

### After Production
1. ✅ Tool usage validation shows 0% skip rate (in logs)
2. ✅ Cache hit rate > 70% with session fairness
3. ✅ Time validation catches > 95% of invalid time usage
4. ✅ User feedback: no decrease in quality due to validation

---

**Fix Plan Complete** ✅
