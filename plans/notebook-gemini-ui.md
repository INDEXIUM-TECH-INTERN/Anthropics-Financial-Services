# NotebookLM + Gemini Hybrid UI Redesign

## 1. Objective
Transform the current "Open Design" (developer-centric, dark/neon) UI into a clean, premium "Google-style" interface that blends the best of **NotebookLM** (document-centric workspace) and **Gemini** (soft aesthetics, high-end typography, floating capsule input).

## 2. Key Files & Context
- `frontend/index.html`: Needs structural updates to match Google's modern web app layouts.
- `frontend/style.css`: Requires a complete aesthetic overhaul (color palette, typography, shadows, border radii).
- `frontend/app.js`: Needs minor tweaks to how sources are injected to match Gemini's source cards.

## 3. UX/UI Analysis & Proposed Changes

### A. Layout Structure (NotebookLM Inspired)
- **Left Sidebar (Sources):** A dedicated area for document management. Source cards will have soft hover effects and clear icons, resembling NotebookLM's source list.
- **Center Area (Chat):** The main focus. Spacious, distraction-free. 
- **Right Sidebar (Studio/Metrics):** A clean, togglable (on large screens) panel for live metrics and logs, styled like a clean system console rather than a hacker terminal.

### B. Visual Language (Gemini Inspired)
- **Typography:** `Google Sans` (primary) and `Inter` (fallback). This instantly gives the "Google AI" feel.
- **Color Palette:** 
  - Backgrounds: Clean white (`#ffffff`) and soft off-white (`#f0f4f9` - Gemini surface color).
  - Text: Deep gray/black (`#1f1f1f`).
  - Accents: Subtle Google Blue (`#0b57d0`).
- **Chat Bubbles:**
  - User: Soft light blue/gray bubble.
  - Bot: No bubble (transparent background), clean text layout, with a subtle "sparkle" or bot icon beside it.
- **Input Area:** A floating, rounded capsule (pill shape) with a subtle shadow, containing the text area and action buttons (attachment, send).

### C. Source Citations (Hybrid)
- Sources will be displayed as clean, clickable "chips" at the bottom of the AI's response, similar to how Gemini cites its sources.

## 4. Implementation Steps

1. **Step 1: HTML Refactor:**
   - Remove `od-theme` and `od-layout` classes.
   - Set up the HTML skeleton for the new 3-column Google Material 3 layout.
   - Update icons to match Google's Material Symbols / Lucide equivalent.

2. **Step 2: CSS Overhaul:**
   - Define CSS variables for the Gemini light theme (`--surface-container`, `--on-surface`, `--primary`).
   - Style the workspace flexbox.
   - Style the chat pane and floating input capsule.
   - Style the source cards with soft shadows (`box-shadow: 0 1px 2px 0 rgba(0,0,0,0.3)` style).

3. **Step 3: JS Enhancements:**
   - Update `app.js` to render the inline sources as Gemini-like citation chips.
   - Add a subtle loading animation (Gemini's glowing gradient or subtle pulse).

## 5. Verification
- After implementation, the UI will be launched and reviewed against the mental models of Gemini Advanced and NotebookLM to ensure a premium, polished feel.
