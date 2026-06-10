# Agent System & Routing

> 10 specialist financial agents, AI-powered routing, and slash commands.

## Agent Catalog

| Agent | Purpose | Skills |
|-------|---------|--------|
| `pitch-agent` | Pitch decks, presentations | pitch-deck, datapack-builder, cim-builder, teaser, buyer-list, comps-analysis, precedent-transactions, lbo-model, merger-model |
| `meeting-prep-agent` | Meeting preparation | briefing-pack, biography-generator, company-profile, news-digest |
| `market-researcher` | Market research | sector-overview, competitive-analysis, comps-analysis, idea-generation, thesis-tracker, catalyst-calendar |
| `earnings-reviewer` | Earnings report analysis | earnings-analysis, earnings-preview, initiating-coverage, model-update, morning-note, xlsx-author |
| `model-builder` | Financial modeling | dcf-model, lbo-model, 3-statement-model, merger-model, xlsx-author, audit-xls |
| `valuation-reviewer` | Valuation review | valuation-review, gp-reporting, lp-reporting |
| `gl-reconciler` | General ledger reconciliation | break-detection, root-cause-analysis, sign-off-routing |
| `month-end-closer` | Month-end closing | accruals, roll-forwards, variance-commentary |
| `statement-auditor` | Statement auditing | lp-statement-audit, distribution-verification |
| `kyc-screener` | KYC screening | onboarding-doc-parsing, gap-flagging |

## Routing Strategy

### Tier 1: AI-Powered Router

The system sends the user query + agent catalog to the LLM with a router prompt. The LLM returns a JSON route plan:

```json
{
  "agent": "earnings-reviewer",
  "skills": ["earnings-analysis"],
  "temporal": {
    "intent": "historical",
    "resolved_date": "2025-06-30",
    "is_future": false
  },
  "reason": "User asked about H1 2025 earnings"
}
```

### Tier 2: Heuristic Fallback

If the AI router fails (LLM error, unparseable JSON), the system falls back to Vietnamese keyword matching:

| Keywords | Target Agent |
|----------|--------------|
| "ban lanh dao", "lanh dao", "board of directors" | `meeting-prep-agent` |
| "doanh thu", "loi nhuan", "quy", "bao cao thu nhap" | `earnings-reviewer` |
| "dinh gia", "dcf", "lbo", "du phong" | `model-builder` |
| "so sanh", "phan tich ky thuat", "gia co phieu", "nganh" | `market-researcher` |
| "kiem toan", "audit", "nam o trang nao" | `statement-auditor` |

### Temporal Resolution

| Expression | Intent | Resolved Date |
|------------|--------|---------------|
| "hom nay", "hien tai" | `realtime` | Today |
| "hom qua" | `latest` | Yesterday |
| "thu hai vua roi" | `historical` | Last Monday |
| "ngay mai", "sap toi", "tuong lai" | `is_future` | — |
| "nam 2024" | `historical` | 2024-12-31 |
| "6 thang dau nam 2025" | `historical` | 2025-06-30 |

### Sanitization

After routing, `sanitizeRoutePlan()` validates:
1. Agent is in the `allowedAgents` whitelist (prevents hallucinated agents)
2. Agent document exists on GitHub
3. Skills are filtered to valid ones per agent
4. Falls back to default skills if none are valid

## Slash Commands

Bypass routing entirely with predefined commands:

| Command | Target Agent | Skills |
|---------|--------------|--------|
| `/earnings <ticker>` | `earnings-reviewer` | `earnings-analysis` |
| `/market <query>` | `market-researcher` | `sector-overview` |

## Agent Handoff

During a conversation, the AI can delegate to another agent via the `handoff_request` tool. The orchestrator picks up the handoff plan on the next ReAct loop iteration, loads the new agent's configuration, bootstraps new context, and continues the conversation with the new agent's context.
