# Chatbox UI Simulation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Xây dựng giao diện Chatbox chuyên nghiệp tích hợp với lõi xử lý tài chính hiện có.

**Architecture:** Chuyển đổi Go CLI thành REST API server và xây dựng Frontend React độc lập. Sử dụng SSE (Server-Sent Events) hoặc REST Polling để mô phỏng luồng log từ Orchestrator.

**Tech Stack:** Go (Backend), React (TypeScript), Vite, Vanilla CSS.

---

### Task 1: Nâng cấp Backend Go thành Web Server

**Files:**
- Modify: `Gemini/cmd/gemini-cli/main.go`
- Create: `Gemini/cmd/gemini-cli/server.go`

- [ ] **Step 1: Khởi tạo Server và Route API**
    Tạo file `Gemini/cmd/gemini-cli/server.go` với các handler cơ bản cho `/api/chat`.

- [ ] **Step 2: Cập nhật Main để hỗ trợ chế độ Server**
    Sửa `Gemini/cmd/gemini-cli/main.go` để nhận flag `--server`.

- [ ] **Step 3: Xử lý CORS và Request/Response**
    Đảm bảo Frontend có thể gọi API mà không bị chặn.

### Task 2: Khởi tạo Frontend React (Vite)

**Files:**
- Create: `Gemini/frontend/` (Vite project)

- [x] **Step 1: Scaffold dự án Vite**
    Chạy lệnh `npm create vite@latest frontend -- --template react-ts`.
- [x] **Step 2: Cài đặt CSS cơ bản cho Chatbox**
    Tạo layout hiện đại giống Claude.


### Task 3: Phát triển UI Components

**Files:**
- Create: `Gemini/frontend/src/components/ChatBox.tsx`

- [ ] **Step 1: Thiết kế tin nhắn (Message bubble)**
    Hiển thị phân biệt User, AI và Logs.

- [ ] **Step 2: Hiệu ứng gõ chữ (Typing Effect)**
    Giả lập stream response.

### Task 4: Kết nối Frontend & Backend

- [ ] **Step 1: Fetch API**
    Gửi tin nhắn từ Frontend lên Backend.

- [ ] **Step 2: Hiển thị MCP Tool Logs**
    Đồng bộ các thông báo `🎯 [Action]` lên giao diện.
