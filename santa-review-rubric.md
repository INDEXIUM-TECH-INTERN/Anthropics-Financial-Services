# Santa Loop Rubric - Agent Response Testing

## Overview
Adversarial dual-review convergence loop testing for agent response optimization, focusing on content accuracy and response time performance.

## Review Criteria

| Criterion | Pass Condition |
|-----------|---------------|
| Correctness | Response logic is sound, handles financial queries correctly, no factual errors in calculations or data interpretation |
| Security | No secrets leaked, proper input validation, safe tool execution, financial data protection |
| Error Handling | Errors handled gracefully with user-friendly messages, no silent failures, retry logic where appropriate |
| Completeness | All requirements addressed, comprehensive responses to financial queries, proper depth for complex topics |
| Performance | Response time under 30 seconds for simple queries, under 2 minutes for complex calculations, no timeout issues |
| Internal Consistency | No contradictions between frontend/backend responses, consistent API formats, coherent conversation flow |
| Time Accuracy | Response time measurements accurate, latency properly reported, performance optimizations working |
| Data Integrity | Financial calculations precise, no rounding errors in monetary values, correct data formatting |

## File Scope
- Go backend: Agent and Orchestrator core components
- Frontend: Chat page streaming implementation
- Related files: Prompt system, context management, streaming logic

## Expected Behavior
1. **Simple queries** (greetings, basic info): < 5 seconds
2. **Moderate queries** (single tool calls): 5-15 seconds  
3. **Complex queries** (multi-tool, calculations): 15-60 seconds
4. **Streaming** should show progress indicators
5. **Error recovery** should be robust and informative

## Testing Scenarios
1. Quick greeting response time
2. Single tool call (search/calculate)
3. Multi-step reasoning with context
4. Streaming response quality
5. Error scenarios (network timeout, API failure)
6. Performance under load

## Review Process
Both reviewers must independently evaluate all criteria against implementation, returning structured feedback.

---

**Return JSON structure:**
```json
{
  "verdict": "PASS" | "FAIL",
  "checks": [
    {"criterion": "Correctness", "result": "PASS|FAIL", "detail": "..."},
    {"criterion": "Security", "result": "PASS|FAIL", "detail": "..."},
    {"criterion": "Error Handling", "result": "PASS|FAIL", "detail": "..."},
    {"criterion": "Completeness", "result": "PASS|FAIL", "detail": "..."},
    {"criterion": "Performance", "result": "PASS|FAIL", "detail": "..."},
    {"criterion": "Internal Consistency", "result": "PASS|FAIL", "detail": "..."},
    {"criterion": "Time Accuracy", "result": "PASS|FAIL", "detail": "..."},
    {"criterion": "Data Integrity", "result": "PASS|FAIL", "detail": "..."}
  ],
  "critical_issues": ["..."],
  "suggestions": ["..."]
}
```