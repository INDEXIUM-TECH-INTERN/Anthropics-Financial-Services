# Frontend React (Vite) Initialization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Scaffold a Vite project and set up the basic Chat UI layout.

**Architecture:** React SPA with TypeScript and Vanilla CSS.

**Tech Stack:** React, TypeScript, Vite, Vanilla CSS.

---

### Task 1: Scaffold Vite Project

**Files:**
- Create: `Gemini/frontend/`

- [x] **Step 1: Run Vite creation command**

Run: `npm create vite@latest frontend -- --template react-ts` in `Gemini/` directory.

- [x] **Step 2: Install dependencies**

Run: `npm install` in `Gemini/frontend/`.

- [x] **Step 3: Verify basic project structure**

Run: `ls -R Gemini/frontend/`
Expected: `src/`, `public/`, `package.json`, `tsconfig.json`, etc.

### Task 2: Setup CSS and Layout

**Files:**
- Modify: `Gemini/frontend/src/App.tsx`
- Modify: `Gemini/frontend/src/App.css`
- Modify: `Gemini/frontend/src/index.css`

- [x] **Step 1: Reset styles in index.css**

```css
:root {
  font-family: Inter, system-ui, Avenir, Helvetica, Arial, sans-serif;
  line-height: 1.5;
  font-weight: 400;
  color-scheme: light dark;
  color: rgba(255, 255, 255, 0.87);
  background-color: #242424;
}

body {
  margin: 0;
  display: flex;
  place-items: center;
  min-width: 320px;
  min-height: 100vh;
}
```

- [x] **Step 2: Define Chat Layout in App.css**

```css
.chat-container {
  display: flex;
  flex-direction: column;
  height: 100vh;
  width: 100vw;
  max-width: 800px;
  margin: 0 auto;
  background-color: #ffffff;
  color: #333333;
}

.chat-header {
  padding: 1rem;
  border-bottom: 1px solid #e5e5e5;
  text-align: center;
  font-weight: bold;
}

.message-list {
  flex: 1;
  overflow-y: auto;
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.message {
  padding: 0.75rem 1rem;
  border-radius: 1rem;
  max-width: 70%;
}

.user-message {
  align-self: flex-end;
  background-color: #007bff;
  color: white;
}

.ai-message {
  align-self: flex-start;
  background-color: #f0f0f0;
  color: #333333;
}

.input-area {
  padding: 1rem;
  border-top: 1px solid #e5e5e5;
  display: flex;
  gap: 0.5rem;
}

.input-area input {
  flex: 1;
  padding: 0.5rem;
  border: 1px solid #ccc;
  border-radius: 4px;
}

.input-area button {
  padding: 0.5rem 1rem;
  background-color: #007bff;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}
```

- [x] **Step 3: Implement App.tsx structure**

```tsx
import React, { useState } from 'react'
import './App.css'

interface Message {
  id: number;
  text: string;
  sender: 'user' | 'ai';
}

function App() {
  const [messages, setMessages] = useState<Message[]>([
    { id: 1, text: "Hello! How can I help you today?", sender: 'ai' }
  ]);
  const [input, setInput] = useState('');

  const handleSend = () => {
    if (!input.trim()) return;
    const newMessage: Message = { id: Date.now(), text: input, sender: 'user' };
    setMessages([...messages, newMessage]);
    setInput('');
  };

  return (
    <div className="chat-container">
      <header className="chat-header">
        Gemini AI Finance
      </header>
      <main className="message-list">
        {messages.map(msg => (
          <div key={msg.id} className={`message ${msg.sender === 'user' ? 'user-message' : 'ai-message'}`}>
            {msg.text}
          </div>
        ))}
      </main>
      <footer className="input-area">
        <input 
          type="text" 
          value={input} 
          onChange={(e) => setInput(e.target.value)} 
          onKeyPress={(e) => e.key === 'Enter' && handleSend()}
          placeholder="Type a message..."
        />
        <button onClick={handleSend}>Send</button>
      </footer>
    </div>
  )
}

export default App
```

### Task 3: Verification

- [x] **Step 1: Build the project**

Run: `npm run build` in `Gemini/frontend/`.

- [x] **Step 2: Check for lint/type errors**

Run: `npm run lint` or `tsc` if available.
