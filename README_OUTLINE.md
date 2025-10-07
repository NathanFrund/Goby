# Goby README.md - Proposed Outline

This document outlines a new structure for the main `README.md` file. We can work through each section one at a time to ensure the documentation is accurate and clear.

---

### 1. Project Title & Introduction

- Project name and logo.
- A brief one-sentence description of what Goby is.

### 2. Quick Start

- The absolute fastest way to get the project running for development.
- Should point to the `make dev` command.
- Briefly mention what this command does (starts Go server and Tailwind watcher).
- Link to the running application.

### 3. Prerequisites

- A clear list of required tools (Go, Node.js, Air, Overmind, tmux).
- Include installation commands for the Go tools.

### 4. Development Workflow

- Explain the recommended `overmind start` workflow.
- Provide the alternative of running `npm run dev:tailwind` and `air` in separate terminals.

### 5. Core Architecture

This is the most important section to update.

#### 5.1. Module System

- Explain the purpose of modules (self-contained features).
- Show the directory structure of a typical module.
- Detail the `module.Module` interface and what each method does (`Name`, `Register`, `Boot`).
- Provide a clear, up-to-date code example of a simple module implementation.
- Explain how to activate a new module in `internal/server/kernel.go`.

#### 5.2. Real-time Architecture (Watermill & WebSockets)

- High-level overview: Explain that the system uses a message bus (Watermill) connected to clients via a WebSocket bridge.
- **The Flow (Broadcast):**
  1. Backend event occurs.
  2. Module renders an HTML fragment.
  3. Module publishes the fragment to a Watermill topic (e.g., `html-broadcast`).
  4. The WebSocket bridge receives the message and forwards it to all connected clients on the `/app/ws/html` endpoint.
  5. htmx on the client-side swaps the content.
- **The Flow (Direct Message):**
  1. Explain that direct messaging is achieved by publishing to a user-specific topic (e.g., `html-direct-user:xyz`).
  2. The bridge routes this message only to the specified user's connections.
- Provide a clear code example (like the wargame engine).

### 6. Template & Asset Management

- Explain the two sources of templates: shared (`web/src/templates`) and module-specific (`internal/modules/...`).
- Clarify how the `APP_TEMPLATES` environment variable (`disk` vs. `embed`) controls loading.
- Explain how static assets (`APP_STATIC`) are handled similarly.

### 7. Production & Deployment

- How to create a self-contained production binary (`make build-embed`).
- List the required runtime environment variables.
- Provide an example systemd service unit file.

### 8. Configuration Reference

- A clear, sectioned list of all environment variables (`.env` and `.env.test`).

### 9. Testing

- How to run the test suite.
- Mention the purpose of `.env.test`.
- Explain the difference between unit tests and the integration tests we just built.
