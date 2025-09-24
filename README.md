# Goby

<p align="center">
  <img src="web/static/img/logo.svg" alt="Goby Mascot" width="200">
</p>

Goby is a project template for building web applications with Go and Tailwind CSS, featuring live-reloading for a great developer experience.

## Quick Start

Get up and running fast.

```sh
# Install JS deps (first run)
npm install

# Development with live reload (recommended)
make dev

# Alternatively, run directly with disk templates
make run

# Run with embedded templates (production-like)
make run-embed

# Build production assets and binary (disk templates unless APP_TEMPLATES=embed at runtime)
make build

# Build production assets and binary preferring embedded templates
make build-embed
```

Once the app is running, open `http://localhost:8080`, log in, and navigate to `Chat`.

- Click "Trigger Hit Event" in the Game State Monitor to see:
  - An HTML fragment injected into the chat via `/app/ws/html`.
  - A JSON update displayed by the monitor via `/app/ws/data`.

## Prerequisites

Before you begin, ensure you have the following tools installed:

- **Go**: Version 1.22 or newer.
- **Node.js and npm**: For managing Tailwind CSS.
- **[Air](https://github.com/air-verse/air)**: For live-reloading the Go application.
- **[Overmind](https://github.com/DarthSim/overmind)**: For running multiple processes (Go and Tailwind) concurrently.
- **[tmux](https://github.com/tmux/tmux/wiki)**: The terminal multiplexer used by `overmind` to manage processes.

### Tool Installation

You can install the required Go and system tools with these commands:

```sh
# Install tmux
# On macOS with Homebrew:
brew install tmux
# On Debian/Ubuntu:
# sudo apt-get install tmux

# Install Air for Go live-reloading
go install github.com/air-verse/air@latest

# Install Overmind process manager
go install github.com/DarthSim/overmind/v2@latest
```

## Development

This project is configured for a streamlined development experience. The recommended approach uses `overmind` to manage both the Go and Tailwind processes from a single command.

1.  **Install dependencies:**

    ```sh
    npm install
    ```

2.  **Start the development server:**

    ```sh
    overmind start
    ```

This will start both processes defined in the `Procfile`. Your application will be available at `http://localhost:8080` and will automatically reload when you make changes to Go or CSS files.

### Alternative: Running in Separate Terminals

If you prefer not to install `overmind` and `tmux`, you can run the Go live-reloader and the Tailwind CSS watcher in two separate terminal shells.

1.  **Terminal 1: Start the Tailwind watcher:**

    ```sh
    npm run dev:tailwind
    ```

2.  **Terminal 2: Start the Go application with Air:**
    ```sh
    air
    ```

This setup achieves the same result, with your Go application running on `http://localhost:8080` and live-reloading enabled for both backend and frontend changes.

## Module System

Goby features a modular architecture inspired by frameworks like Laravel. Features are organized into self-contained packages under `internal/modules/`. Each module implements the `module.Module` interface, allowing it to be registered and booted by the core application. This promotes strong separation of concerns and makes the application highly extensible.

### Creating a New Module

Follow these steps to create a new module:

1.  **Create the Module Structure**
    Create a new directory for your module under `internal/modules/`. For a module named `greeter`, the structure would be:

    ```
    internal/modules/
    └── greeter/
        ├── module.go          # Module interface implementation
        ├── greeter.go         # Core service logic and types
        └── templates/
            └── components/
                └── greeting.html
    ```

2.  **Implement the `Module` Interface**
    In `greeter/module.go`, create a struct (e.g., `GreeterModule`) and implement the `module.Module` interface.

    - `Name()`: Return a unique name for the module (e.g., `"greeter"`).
    - `RegisterTemplates()`: Register any embedded HTML templates with the renderer.
    - `Register()`: Read configuration and register the module's services (e.g., `greeter.Service`) into the service locator.
    - `Boot()`: Retrieve services from the service locator and register the module's HTTP routes.

```go
 // internal/modules/greeter/module.go
 package greeter

 type GreeterModule struct{}

 func (m *GreeterModule) Name() string { return "greeter" }

 func (m *GreeterModule) Register(sl registry.ServiceLocator, cfg config.Provider) error {
     // Create and register your service
     svc := NewService()
     sl.Set("greeter.service", svc)
     return nil
 }

 func (m *GreeterModule) Boot(g *echo.Group, sl registry.ServiceLocator) error {
     // Get your service and register routes
     svc, _ := sl.Get("greeter.service")
     handler := NewHandler(svc.(*Service))
     g.GET("/greet", handler.Greet)
     return nil
 }
 // ... other interface methods
```

3.  **Activate the Module**
    Finally, add your new module to the `AppModules` slice in `internal/server/kernel.go`. This is the single place to enable or disable modules for the entire application.
    ```go
    // internal/server/kernel.go
    var AppModules = []module.Module{
        &wargame.WargameModule{},
        &greeter.GreeterModule{}, // Add your new module here
    }
    ```

## Templates, Modules, and Embedding

Goby supports two template sources to balance fast development and self-contained production builds.

- **Shared templates (layouts/components/pages)** live under `web/src/templates/`.
- **Module templates** live under `internal/modules/<module>/templates/` and are namespaced by the module name.
  - Example: `internal/modules/wargame/templates/components/wargame-damage.html` is rendered as `wargame/wargame-damage.html`.

### How templates are loaded

- **Shared templates** are loaded either from disk or from an embedded filesystem, depending on the `APP_TEMPLATES` environment variable.
  - `APP_TEMPLATES=disk` (default): read from disk via `templates.NewRenderer("web/src/templates")`.
  - `APP_TEMPLATES=embed`: read from the embedded FS defined in `web/src/templates/embed.go` via `templates.NewRendererFromFS(webtemplates.FS, ".")`.
- **Module templates** are registered in two ways:
  - Each module can provide embedded templates (see `internal/modules/wargame/engine.go` and its `RegisterTemplates` function).
  - At startup, the server auto-discovers `internal/modules/*/templates/` directories and registers any templates found from disk, namespaced by the module folder name. This lets disk templates override embedded ones during development.

### Why namespacing?

To avoid collisions between module templates and shared components, module templates are registered under the module name. For example, the `wargame` module registers templates as `wargame/<filename>.html`.

### Running in development (disk templates)

Use disk-based templates for fast iteration:

```sh
make dev          # recommended (Overmind + live-reload)
# or
make run          # simpler: go run with APP_TEMPLATES=disk
```

### Running with embedded templates (production-like)

Validate your production path locally using embedded templates:

```sh
make run-embed    # go run with APP_TEMPLATES=embed
```

### Building for production

Build the Go binary and production CSS (disk or embedded):

```sh
make build        # builds binary (disk templates unless APP_TEMPLATES=embed at runtime)
make build-embed  # builds binary with APP_TEMPLATES=embed set at build time
```

In production, set `APP_TEMPLATES=embed` to force the binary to use embedded templates.

## Production Deployment

This project can produce a self-contained binary that embeds all templates.

- **Static assets** (CSS/JS/images) are served from `web/static/` at runtime and are not embedded. Build them before deploying.
- **Templates** can be embedded via `APP_TEMPLATES=embed`.

### Build steps

```sh
# Build minified CSS and the binary (with embedded templates enabled)
make build-embed
# Or, explicitly
APP_TEMPLATES=embed go build -o ./tmp/goby ./cmd/server
npm run build:js
npm exec tailwindcss -- --input=./web/src/css/input.css --output=./web/static/css/style.css --minify
```

### Runtime environment

Set these environment variables in production (via your process manager or unit file):

- `SERVER_ADDR` (e.g., `:8080`)
- `APP_BASE_URL` (e.g., `https://yourdomain.com`)
- `SESSION_SECRET` (required)
- `SURREAL_URL`, `SURREAL_NS`, `SURREAL_DB`, `SURREAL_USER`, `SURREAL_PASS`
- `EMAIL_PROVIDER` (e.g., `log` or `resend`), `EMAIL_API_KEY`, `EMAIL_SENDER`
- `APP_TEMPLATES=embed` to ensure the binary uses embedded templates.

Example systemd service snippet:

```ini
[Service]
Environment=SERVER_ADDR=:8080
Environment=APP_BASE_URL=https://yourdomain.com
Environment=SESSION_SECRET=change-me
Environment=SURREAL_URL=ws://localhost:8000/rpc
Environment=SURREAL_NS=app
Environment=SURREAL_DB=app
Environment=SURREAL_USER=app
Environment=SURREAL_PASS=secret
Environment=EMAIL_PROVIDER=log
Environment=APP_TEMPLATES=embed
ExecStart=/opt/goby/goby
WorkingDirectory=/opt/goby
# Ensure web/static exists and contains built assets
```

### Deployable artifacts

- Binary: `./tmp/goby`
- Static assets directory: `web/static/`

Make sure `web/static/` (including `css/style.css`) is deployed alongside your binary or served via a CDN.

## Real-time Architecture: The Presentation-Centric Hub

A core feature of this template is its real-time architecture, designed for modularity and scalability. It's built around a central "hub" that acts as a distribution channel for pre-rendered HTML fragments.

This "presentation-centric" approach allows various backend services (e.g., a chat module, a game engine, a notification service) to operate independently. They can focus on their own logic, render their state into a self-contained HTML component, and then publish it to the hub for delivery to all connected clients.

### The Flow

The data and presentation flow follows these steps:

1.  **Event Occurs:** An event is triggered somewhere in the backend. This could be a user sending a chat message or a game engine calculating a state change.
2.  **Render Fragment:** The service responsible for the event uses the application's template renderer to create a self-contained HTML fragment representing the new state (e.g., a `<div>` for a new chat message). This fragment often includes `hx-swap-oob` attributes to tell htmx where to place it on the client-side.
3.  **Publish to Hub:** The service sends the fully rendered HTML fragment (as a `[]byte`) to the central hub's broadcast channel.
4.  **Hub Broadcasts:** The hub receives the HTML fragment and immediately distributes it to every connected WebSocket client.
5.  **Client Receives & Swaps:** The client's browser receives the HTML fragment over the WebSocket connection. htmx processes the fragment, sees the `hx-swap-oob` attribute, and swaps the content into the correct place in the DOM.

### Example: Wargame Engine

Imagine a tabletop game engine running on the server. When one unit damages another, the engine can publish this event to all observers.

1.  The `WargameEngine` calculates that "Alpha Squad" takes 15 damage.
2.  It uses the renderer to create an HTML component from a hypothetical `wargame-damage.html`:
    ```html
    <div hx-swap-oob="beforeend:#game-log">
      <div class="p-2 text-red-500">Alpha Squad takes 15 damage!</div>
    </div>
    ```
3.  The engine sends this HTML to `hub.Broadcast`.
4.  All connected clients receive the fragment, and htmx appends it to the element with the ID `#game-log`.

This architecture decouples the game engine from the complexities of WebSocket and client management, allowing for clean, modular, and highly scalable real-time features.

This project includes a live, interactive demonstration of this feature. Once logged in, navigate to the **Chat** page. You will find a "Game State Monitor" with a button to trigger a random wargame event. Clicking it will publish an HTML fragment to the chat log and a JSON data object to the monitor, showcasing both real-time channels in action.

### Direct Messaging to Specific Users

In addition to broadcasting to all clients, the hub supports sending direct messages to a specific user, even if they have multiple connections (e.g., on a desktop and a phone).

This is achieved by sending a `hub.DirectMessage` struct to the `hub.Direct` channel.

#### The Flow for Direct Messages

1.  **Event Occurs:** A backend service determines that a specific user needs to receive a private notification.
2.  **Render Fragment:** The service renders the appropriate HTML fragment for the notification.
3.  **Create Direct Message:** The service creates a `hub.DirectMessage` struct, populating the `UserID` of the recipient and the `Payload` with the rendered HTML.
4.  **Publish to Direct Channel:** The message is sent to the `hub.Direct` channel.
5.  **Hub Routes Message:** The hub looks up all active connections for the specified `UserID` and sends the payload only to them.

#### Example: Private Notification

If a player's unit is hit, the wargame engine can send them a private alert that appears at the top of their screen.

1.  The engine identifies the `UserID` of the player whose unit was hit.
2.  It renders a `wargame-hit-notification.html` component.
3.  It creates and sends the `DirectMessage`:
    ```go
    directMessage := &hub.DirectMessage{
        UserID:  "user:some_user_id",
        Payload: renderedHTML,
    }
    engine.hub.Direct <- directMessage
    ```
4.  Only the user with the ID `user:some_user_id` will receive the notification.

## Configuration

The application is configured using environment variables. For local development, you can create a `.env` file in the project root to manage these settings.

### Database

- `SURREAL_URL`: The URL of your SurrealDB instance (e.g., `ws://localhost:8000/rpc`).
- `SURREAL_NS`: The namespace to use in SurrealDB.
- `SURREAL_DB`: The database to use in SurrealDB.
- `SURREAL_USER`: The user for authenticating with SurrealDB.
- `SURREAL_PASS`: The password for authenticating with SurrealDB.

### Server

- `SERVER_ADDR`: The address and port for the server to listen on. Defaults to `:8080`.
- `APP_BASE_URL`: The public base URL for the application, used for generating links in emails. Defaults to `http://localhost:8080` for local development.

### Email

- `EMAIL_PROVIDER`: The email service to use. Defaults to `log` (which prints emails to the console). Set to `resend` to use the Resend API.
- `EMAIL_API_KEY`: Your API key for the chosen email provider (e.g., your Resend API key).
- `EMAIL_SENDER`: The "from" address for outgoing emails (e.g., `you@yourdomain.com`). For Resend, this can be omitted to use the default `onboarding@resend.dev`.

### Testing

Integration tests require a running test database. Configuration for tests is managed in a separate `.env.test` file in the project root. You can copy your `.env` file to get started:

```sh
cp .env .env.test

```
