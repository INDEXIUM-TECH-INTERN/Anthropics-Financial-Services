# How to Run the Anthropics-Financial-Services Program

This folder contains a **Financial AI Agent** built in Go (Gemini backend) + modern web frontend (Indexium workspace).

## Primary Program: Gemini Agent + Web UI (Recommended)

### Prerequisites
- Go is installed on this machine (`go version` shows go1.25+).
- API Keys (free tiers available):
  1. **Gemini API Key** (required for core): https://aistudio.google.com/app/apikey
  2. **OpenRouter API Key(s)** (strongly recommended for model fallbacks + free options): https://openrouter.ai/keys
  3. **SerpAPI Key** (optional, for live market search tool): https://serpapi.com/

### Step-by-step (PowerShell)

1. Open **PowerShell** (or Windows Terminal).

2. Navigate to the root repository folder:
   ```powershell
   cd /path/to/Anthropics-Financial-Services
   ```

3. Prepare your API keys in the `Gemini/` folder (first time):
   ```powershell
   cd Gemini
   copy .env.example .env
   notepad .env     # Edit and paste your real keys, save & close
   cd ..            # Go back to root
   ```

4. Run the full program (Web UI + backend) using the root helper:
   ```powershell
   .\run-server.ps1
   ```

5. Open browser:
   - Go to **http://localhost:8080**

6. Use the chat interface:
   - Type financial questions (Vietnamese or English).
   - Example chips are pre-filled for banking/finance analysis.
   - Click **"CHẠY BỘ KIỂM THỬ TỰ ĐỘNG"** to run the evaluator tests.

### Alternative: Pure CLI mode (no UI)

```powershell
# One-shot query using the unified launcher
.\run-server.ps1 -Query "So sánh tổng tài sản HDB và ACB trong 3 năm gần đây"
```

### Notes
- The server auto-detects and serves the `frontend` folder.
- All prompts, routing logic, tools (research/scrape/calculate) live under `Gemini/internal/`.
- No external Go dependencies (pure stdlib + direct HTTP calls).
- SSE is used for live "execution plan" updates in the right sidebar.

## Quick Troubleshooting

- "port already in use": Kill previous instance or change port in `Gemini/internal/api/server.go`.
- No response / auth error: Check `.env` is loaded (server prints "Loading environment from:..."), or use settings modal. Gemini key must be in env at startup.
- Frontend not loading: Confirm the "Serving static files from: ../frontend" message on start.
- Vietnamese UI: The whole workspace is localized for Vietnamese financial research.

Enjoy using the Indexium Financial AI Agent!

