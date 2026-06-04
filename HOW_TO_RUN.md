# How to Run the Anthropics-Financial-Services Program

This folder contains a **Financial AI Agent** built in Go (Gemini backend) + modern web frontend (Indexium workspace).

## Primary Program: Gemini Agent + Web UI (Recommended)

Location of code: `Gemini/` subfolder (the full-featured version with orchestrator, tools, multi-provider).

### Prerequisites
- Go is already installed on this machine (`go version` shows go1.25+).
- API Keys (free tiers available):
  1. **Gemini API Key** (required for core): https://aistudio.google.com/app/apikey
  2. **OpenRouter API Key(s)** (strongly recommended for model fallbacks + free options): https://openrouter.ai/keys
  3. **SerpAPI Key** (optional, for live market search tool): https://serpapi.com/

### Step-by-step (PowerShell)

1. Open **PowerShell** (or Windows Terminal).

2. Navigate to the Gemini code:
   ```powershell
   cd 'C:\indexium\Term 2\Anthropics-Financial-Services\Gemini'
   ```

3. Prepare your API keys (first time):
   ```powershell
   # Copy the template
   copy .env.example .env
   notepad .env     # Edit and paste your real keys, save & close
   ```

4. Run the full program (Web UI + backend):
   ```powershell
   # Option A: Use the helper (easiest)
   .\run.ps1

   # Option B: Direct
   go run cmd/gemini-cli/main.go -server
   ```

5. Open browser:
   - Go to **http://localhost:8080**

6. Use the chat interface:
   - Type financial questions (Vietnamese or English).
   - Example chips are pre-filled for banking/finance analysis.
   - Click the **gear icon** (settings) to update OpenRouter keys live.
   - Click **"CHẠY BỘ KIỂM THỬ TỰ ĐỘNG"** to run the built-in evaluator tests.

### Alternative: Pure CLI mode (no UI)

```powershell
cd 'C:\indexium\Term 2\Anthropics-Financial-Services\Gemini'

# Interactive loop
go run cmd/gemini-cli/main.go

# One-shot question
go run cmd/gemini-cli/main.go "So sánh tổng tài sản HDB và ACB trong 3 năm gần đây"
```

### Other Programs in the Folder (less complete / example only)

- `Claude/main.go` : One-off direct call to Anthropic Claude. Hardcoded paths to another user's repo. Requires `ANTHROPIC_API_KEY`. Needs editing to be useful.

- `..\Anthropics-Financial-Services-main\main.go` : Standalone orchestrator that picks agent+skill then calls Gemini. Also has hardcoded repo path (`C:\Users\Rabuno\...`). Run with `go run main.go` (after fixing paths or having the skills/agents markdowns).

### Notes
- The server auto-detects and serves `../frontend` when launched from inside `Gemini/`.
- All prompts, routing logic, tools (research/scrape/calculate) live under `internal/`.
- No external Go dependencies (pure stdlib + direct HTTP calls).
- SSE is used for live "execution plan" updates in the right sidebar.

## Quick Troubleshooting

- "port already in use": Kill previous instance or change port in `internal/api/server.go`.
- No response / auth error: Check `.env` is loaded (server prints "Loading environment from:..."), or use settings modal for OR keys. Gemini key must be in env at startup.
- Frontend not loading: Confirm the "Serving static files from: ../frontend" message on start.
- Vietnamese UI: The whole workspace is localized for Vietnamese financial research.

Enjoy using the Indexium Financial AI Agent!
