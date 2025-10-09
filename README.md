# Goby

<p align="center">
  <img src="web/static/img/logo.svg" alt="Goby Mascot" width="150">
</p>

Combine Go's performance with modern web development practices to create responsive, component-based web applications that scale.

## Tech Stack

- **Backend**: Go 1.22+
- **Frontend**: HTMX, Templ, and Gomponents
- **Database**: SurrealDB
- **Real-time**: WebSockets with custom bridge
- **Messaging**: Watermill for event-driven architecture
- **Development**: Seamless development experience with Overmind orchestrating Go hot-reloading (Air), template compilation (Templ), and CSS processing (Tailwind).

## Quick Start

Get started with Goby in minutes. This section will help you set up your development environment and run the application.

## Prerequisites

Before you begin, ensure you have the following tools installed:

- **Go** (1.22 or newer) - [Download](https://golang.org/dl/)
- **Node.js and npm** (LTS version recommended) - [Download](https://nodejs.org/)
- **Overmind** - Process manager for development

  ```sh
  # Install with Go (recommended)
  go install github.com/DarthSim/overmind/v2@latest

  # Alternative installations:
  # macOS (Homebrew): brew install overmind
  ```

## Installation

1. **Clone the repository**

   ```sh
   git clone https://github.com/yourusername/goby.git
   cd goby
   ```

2. **Install Node.js dependencies**

   ```sh
   npm install
   ```

## Running the Application

### Using Overmind (Recommended)

Goby uses [Overmind](https://github.com/DarthSim/overmind) to manage multiple processes during development with a single command:

```sh
make dev
```

This starts all required processes defined in the `Procfile`:

- Go application with hot-reloading (via `air`)
- Templ file watcher
- Tailwind CSS compiler
- Other development services

### Alternative: Manual Process Management

If you prefer not to use Overmind, you can run processes separately:

```sh
# Terminal 1: Start the Go application with air
air

# Terminal 2: Watch for template changes
templ generate --watch

# Terminal 3: Start the Tailwind CSS compiler
npm run dev:tailwind
```

## Why Choose Goby?

Goby is built around a presentation-first architecture that makes building modern, real-time web applications a joy. Here's what sets it apart:

### Beautiful UIs by Default

- Pre-configured with Tailwind CSS and DaisyUI components
- Stunning, accessible interfaces out of the box
- Consistent design system that's easy to customize

### Component-Based Development

- Build with **Templ** and **Gomponents** for type-safe, reusable UI components
- Server-rendered components with automatic DOM updates
- Compose complex UIs from simple, focused components

### Real-Time by Design

- Built-in WebSocket support with automatic DOM updates
- Seamless real-time user experiences
- Automatic state synchronization between server and client

### Developer Experience First

- Hot-reloading for both Go and frontend assets
- Comprehensive tooling that stays out of your way
- Clear project structure and sensible defaults

### Flexible Architecture

- Serve both HTML and JSON from the same endpoints
- Event-driven architecture with Watermill message bus
- Modular design that grows with your application

## Exploring the Framework

1. Open `http://localhost:8080` in your browser
2. Log in.
3. Navigate to the **Chat** page to see real-time features in action:
   - Click "Trigger Hit Event" in the Game State Monitor
   - Watch HTML fragments update in the chat log via WebSocket
   - See raw JSON data update in the monitor

## Next Steps

- Check out the [Module System](#module-system) section to learn how to extend Goby
- Explore the example modules in the `internal/modules` directory
- Review the [Configuration](#configuration) section for environment variables and settings

### Architecture Overview

Goby's architecture is designed to make building modern web applications straightforward and maintainable. At its core, it combines the performance of Go with modern web development practices to deliver fast, responsive user experiences.

### Core Architecture

1. **UI-First Design**

   - Backend services are structured around delivering UI components
   - Real-time updates are a first-class concern
   - Components manage their own state and updates

2. **Component-Based Architecture**

   - Build reusable UI components with **Templ** and **Gomponents**
   - Components can be composed together to build complex interfaces
   - Each component can update independently

3. **Real-Time by Default**
   - Built-in WebSocket support for live updates
   - HTMX for fine-grained DOM updates without full page reloads
   - Automatic state synchronization between server and client

### UI-First Design in Practice

Goby's architecture is designed around the concept of delivering complete UI components from the server, enabling a seamless development experience where the UI is not an afterthought but a primary concern.

#### Component Rendering

Components are rendered on the server using **Templ** and **Gomponents**, which generate efficient, type-safe HTML. This approach ensures that:

- UI logic is co-located with business logic
- Components are reusable across different parts of the application

Each component manages its own state and can update independently, making it easy to build complex, interactive UIs without complex client-side state management.

#### Component-Based Architecture

Goby's component system is built on two complementary technologies:

1. **Templ** for type-safe HTML templates
2. **Gomponents** for composable UI components

This combination allows developers to build complex UIs from simple, reusable components while maintaining type safety and good performance characteristics.

#### Component Lifecycle

1. **Definition**: Components are defined as Go types that implement the `templ.Component` interface
2. **Rendering**: Components render themselves to HTML on the server

### Real-Time by Default

Goby's real-time capabilities are built into the core of the framework, making it easy to add live updates to any part of your application.

#### Real-time Communication

Goby uses a message bus (Watermill) connected to clients via WebSockets to enable real-time updates. This architecture allows for efficient communication between the server and clients, whether they're web browsers, mobile apps, or other services.

#### Client Types

Goby supports multiple client types through its flexible architecture:

1. **Web Browsers (HTMX)**

   - Receives pre-rendered HTML fragments
   - Zero client-side JavaScript required for basic interactions
   - Automatic DOM updates via HTMX WebSockets
   - Example WebSocket endpoint: `/ws/html`

2. **Native Mobile/Desktop Apps**
   - Connects via WebSockets or HTTP/2 Server-Sent Events (SSE)
   - Receives structured JSON data instead of HTML
   - Can subscribe to specific data channels
   - Example WebSocket endpoint: `/ws/data`

### Message Flow for Web Clients

1. **Backend Event**: An event occurs in the backend (e.g., a new chat message is posted).
2. **HTML Rendering**: The module renders an HTML fragment using either:

   - **Templ**: Type-safe HTML templates that compile to Go code
   - **Gomponents**: Composable HTML components in pure Go

   You can also use them together - Templ for page layouts and Gomponents for reusable UI components.

3. **Message Publishing**: The module publishes the fragment to a Watermill topic (e.g., `html-broadcast`).
4. **WebSocket Delivery**: The WebSocket bridge receives the message and forwards it to all connected clients on the `/app/ws/html` endpoint.
5. **Client Update**: htmx on the client-side swaps the content into the appropriate part of the page.

### Message Flow for Data Clients

For native mobile/desktop applications that work with raw data:

1. **Backend Event**: An event occurs in the backend (e.g., game state changes).
2. **Data Preparation**: The module prepares structured data (JSON/Protobuf).
3. **Message Publishing**: The module publishes the data to a Watermill topic (e.g., `data-broadcast` or `data-user:user123`).
4. **WebSocket Delivery**: The WebSocket bridge forwards the message to the appropriate clients on the `/app/ws/data` endpoint.
5. **Client Processing**: The native client processes the data and updates its UI or state accordingly.

Example data message structure:

```json
{
  "type": "game_update",
  "data": {
    "game_id": "12345",
    "players": ["player1", "player2"],
    "scores": { "player1": 10, "player2": 8 }
  },
  "timestamp": "2025-10-07T23:14:14Z"
}
```

### Direct Messaging

For user-specific updates:

1. Messages are published to user-specific topics (e.g., `html-direct-user:user123` or `data-user:user123`).
2. The WebSocket bridge routes these messages only to the specified user's active connections.

### Data-First API for Native Clients

For non-HTML clients, Goby provides a clean data API:

```go
// Publish structured data for native clients
payload := map[string]interface{}{
    "type": "game_state_update",
    "data": gameState,
    "timestamp": time.Now().UTC().Format(time.RFC3339),
}

h.publisher.Publish("data-broadcast", message.NewMessage(
    uuid.New().String(),
    payload,
))
```

Native clients can subscribe to specific data channels:

- `data-broadcast` - Public data updates
- `data-user:{userID}` - User-specific updates
- `data-game:{gameID}` - Game-specific updates

### Example: Game State Updates

Here's how the wargame module handles real-time updates:

```go
// Publish a game state update
payload := map[string]interface{}{
    "type": "game_update",
    "data": gameState,
}

// Publish to all connected clients
h.publisher.Publish("html-broadcast", message.NewMessage(
    uuid.New().String(),
    payload,
))
```

### Real-time Architecture: The Watermill Bridge

A core feature of this template is its real-time architecture, designed for modularity and scalability. It's built around a **Watermill** message bus, which is connected to clients via a **WebSocket Bridge**. This allows backend modules to communicate with each other and with the frontend in a decoupled manner.

This "presentation-centric" approach allows various backend services (e.g., a chat module, a game engine) to operate independently. They can focus on their own logic, render their state into a self-contained HTML component, and then publish it to a Watermill topic for delivery to clients.

### The Broadcast Flow

The data and presentation flow follows these steps:

1. **Event Occurs:** An event is triggered somewhere in the backend. This could be a user sending a chat message or a game engine calculating a state change.
2. **Render Fragment:** The responsible module uses the application's template renderer to create a self-contained HTML fragment representing the new state (e.g., a `<div>` for a new chat message). This fragment often includes `hx-swap-oob` attributes to tell htmx where to place it on the client-side.
3. **Publish to Topic:** The module publishes a message containing the rendered HTML to a broadcast topic (e.g., `html-broadcast` or `data-broadcast`).
4. **Bridge Subscribes & Forwards:** The WebSocket Bridge, which subscribes to these topics, receives the message.
5. **Bridge Delivers to Client:** The bridge forwards the message payload to the appropriate WebSocket endpoint (`/app/ws/html` or `/app/ws/data`), delivering it to all connected clients.
6. **Client Receives & Swaps:** The client's browser receives the HTML fragment over the WebSocket connection. htmx processes the fragment, sees the `hx-swap-oob` attribute, and swaps the content into the correct place in the DOM.

### Example: Wargame Engine

Imagine a tabletop game engine running on the server. When one unit damages another, the engine can publish this event to all observers.
This code snippet from the `wargame` module demonstrates the process:

```go
// internal/modules/wargame/service.go

 func (s *Service) handleHit(target string, damage int) error {
     // 1. Create an instance of the compiled `templ` component.
    component := view.WargameDamage(target, damage)

    // 2. Render the component to an HTML string.
    html, err := view.RenderComponent(component)
    if err != nil {
        return err
    }

    // 3. Create a new Watermill message with the rendered HTML.
    msg := message.NewMessage(watermill.NewUUID(), []byte(html))

    // 4. Publish the message to the global broadcast topic.
    return s.pubsub.Publish(topics.HTMLBroadcast, msg)
 }
```

The corresponding `templ` component (`wargame_damage.templ`) defines the structure and uses an `hx-swap-oob` attribute to tell htmx where to place the content:

```templ
templ WargameDamage(target string, damage int) {
    <div hx-swap-oob="beforeend:#game-log">
        <div class="p-2 text-red-500">{ target } takes { fmt.Sprintf("%d", damage) } damage!</div>
    </div>
}
```

This architecture decouples the game engine from the complexities of WebSocket and client management, allowing for clean, modular, and scalable real-time features.

### The Direct Message Flow

In addition to broadcasting, the system supports sending direct messages to a specific user. This is achieved by calling the `SendDirect` method on the WebSocket bridge.

For services that are integrated with the real-time layer (e.g., a subscriber that is initialized with a reference to the bridge), it's straightforward to call the bridge's methods directly. This bypasses the pub/sub system for the outbound message.

```go
// A service with access to the bridge can send a message directly.
bridge.SendDirect(userID, payload, websocket.ConnectionTypeHTML)
```

This project includes a live, interactive demonstration of this feature. Once logged in, navigate to the **Chat** page. You will find a "Game State Monitor" with a button to trigger a random wargame event. Clicking it will publish an HTML fragment to the chat log and a JSON data object to the monitor, showcasing both real-time channels in action.

### UI Components & Asset Management

Goby uses modern web development tools for building and managing UI components and static assets, providing a fast development experience with hot-reloading and optimized production builds.

#### UI Components

Goby leverages two powerful templating systems:

- **[Templ](https://templ.guide/)**: A type-safe HTML templating language for Go that compiles to Go code, providing excellent performance and IDE support.
- **[Gomponents](https://www.gomponents.com/)**: A view library for writing HTML in Go, offering a clean, type-safe way to build UI components.

#### Template Organization

Goby's UI components are organized in the following structure:

- **`web/src/templates/`** - Main templates directory

  - `components/` - Reusable UI components
  - `layouts/` - Base layouts and page templates
  - `pages/` - Page-specific templates
  - `partials/` - Reusable template partials

- **Module Templates** - Feature-specific templates are located in their respective module directories:
  - `internal/modules/<module-name>/templates/`

#### Component Development

1. **Templ Components**:

   - Create `.templ` files for your components
   - Changes are automatically picked up by the `templ generate --watch` process
   - Import and use components in your Go code

2. **Gomponents**:

   - Create Go files that use the `g` package to build UI components
   - Components are just Go functions that return `g.Node`
   - Use them directly in your handlers or other components

3. **Hot Reloading**:
   The development server automatically handles changes to:
   - Go files (via `air`)
   - Templ files (via `templ generate --watch`)
   - CSS/JS (via Tailwind's JIT compiler)

#### Module System

Goby's architecture is built around the concept of modules - self-contained packages that encapsulate related functionality. Each module is responsible for its own routes, services, and UI components, making it easy to add, remove, or modify features without affecting other parts of the application.

#### Core Concepts

1. **Module Structure**

   - Each module lives in its own directory under `internal/modules/`
   - Implements the `module.Module` interface with `Name()` and `Boot()` methods
   - Can include routes, services, templates, and static assets
   - Follows Go's standard package structure and conventions

2. **Type-Safe Dependency Injection**

   - Uses a type-safe `Registry` for dependency injection
   - Services are registered and resolved by their type, not by string keys
   - Provides compile-time safety and better IDE support
   - No need for manual casting or type assertions

3. **Lifecycle**
   - **Boot Phase**: Called once during application startup after all services are registered
     - Set up HTTP routes and handlers
     - Initialize background services and workers
     - Register event handlers and subscriptions
   - **Runtime**: Handles incoming requests and events
   - **Shutdown**: Graceful cleanup (handled automatically by the framework)

### Creating a New Module

Follow these steps to create and integrate a new module:

#### 1. Create the Module Structure

```sh
internal/modules/
  yourmodule/
    ├── handler.go     # HTTP request handlers
    ├── service.go     # Business logic and core functionality
    ├── module.go      # Module definition and lifecycle hooks
    ├── templates/     # Optional: Template files (.templ)
    └── static/        # Optional: Static assets (CSS, JS, images)
```

Key files:

- `module.go`: Implements the `module.Module` interface
- `handler.go`: Defines HTTP request handlers (optional)
- `service.go`: Contains business logic (optional)
- `templates/`: Server-rendered templates (optional)
- `static/`: Static assets (optional)

#### 2. Define Service Interfaces (Recommended)

Define interfaces for your services to enable better testability and dependency injection:

```go
// internal/modules/yourmodule/service.go
package yourmodule

// YourService defines the core functionality of your module
type YourService interface {
    DoSomething() error
}

// Implementation of the service
type serviceImpl struct{}

func NewService() YourService {
    return &serviceImpl{}
}

func (s *serviceImpl) DoSomething() error {
    // Implementation here
    return nil
}
```

#### 3. Implement the Module Interface

Create `module.go` with the following structure:

```go
// internal/modules/yourmodule/module.go
package yourmodule

import (
    "context"
    "log/slog"

    "github.com/labstack/echo/v4"
    "github.com/nfrund/goby/internal/module"
    "github.com/nfrund/goby/internal/registry"
    "github.com/nfrund/goby/internal/pubsub"
)

// YourModule implements the module.Module interface
type YourModule struct {
    module.BaseModule  // Embed BaseModule for common functionality
}

// New creates a new instance of the module
func New() *YourModule {
    return &YourModule{}
}

// Name returns the module's unique identifier
func (m *YourModule) Name() string {
    return "yourmodule"
}

// Boot is called during application startup after all services are registered
func (m *YourModule) Boot(g *echo.Group, reg *registry.Registry) error {
    // Resolve dependencies by their interface types
    publisher := registry.MustGet[pubsub.Publisher](reg)

    // Initialize services
    service := NewService()
    handler := NewHandler(service, publisher)

    // Register routes
    // The server mounts routes under /app/{moduleName}
    g.GET("/endpoint", handler.HandleEndpoint)

    // Start background workers if needed
    go func() {
        subscriber := registry.MustGet[pubsub.Subscriber](reg)
        worker := NewWorker(service, subscriber)
        if err := worker.Start(context.Background()); err != nil {
            slog.Error("Worker failed", "module", m.Name(), "error", err)
        }
    }()

    return nil
}
```

#### 4. Register the Module

Add your module to the application in `cmd/server/main.go`:

```go
modues := []module.Module{
    // Core modules first
    wargame.New(),
    chat.New(),
    yourmodule.New(),  // Add your module here

    // More modules...
}
```

**Module Registration Notes**:

1. **Core Services First**: Core services (database, pub/sub, etc.) are registered in `main.go` before modules
2. **Module Initialization**: Each module provides a constructor (e.g., `New()`) that returns a `module.Module`
3. **Explicit Registration**: Modules are explicitly listed in the `modules` slice in `main.go`

This approach offers several benefits:

- Clear visibility of all active modules in one place
- Simple dependency management through the type-safe registry

#### Best Practices

1. **Module Design**

   - **Single Responsibility**: Each module should focus on one domain concern
   - **Loose Coupling**: Depend on interfaces, not concrete implementations
   - **Encapsulation**: Keep implementation details private to the module
   - **Error Handling**: Return meaningful errors and use custom error types for domain-specific errors

2. **Dependency Management**

   - **Constructor Injection**: Pass dependencies through constructors
   - **Interface Segregation**: Define small, focused interfaces
   - **Lazy Initialization**: Initialize resources only when needed

   ```go
   // Good: Dependencies are explicit and type-safe
   type Handler struct {
       service  YourService
       publisher pubsub.Publisher
   }

   func NewHandler(service YourService, pub pubsub.Publisher) *Handler {
       return &Handler{
           service:  service,
           publisher: pub,
       }
   }
   ```

3. **Concurrency**

   - Use context for cancellation and timeouts
   - Start background workers in Boot() using goroutines
   - Handle graceful shutdown of resources

   ```go
   // Example of a background worker
   func (w *Worker) Start(ctx context.Context) error {
       for {
           select {
           case <-ctx.Done():
               return ctx.Err()
           case msg := <-w.messages:
               if err := w.processMessage(msg); err != nil {
                   slog.Error("Failed to process message", "error", err)
               }
           }
       }
   }
   ```

4. **Testing**

   - Write unit tests for business logic
   - Test HTTP handlers with echo's test utilities
   - Use table-driven tests for different scenarios
   - Mock external dependencies using interfaces

   ```go
   func TestYourHandler(t *testing.T) {
       // Setup
       mockService := &MockService{}
       mockPublisher := &MockPublisher{}
       h := NewHandler(mockService, mockPublisher)

       // Test cases...
   }
   ```

#### Example Module

See `internal/modules/chat` for a complete implementation reference.

### Static Assets

Static assets (CSS, JS, images) are managed in the `web/` directory:

- `web/static/`: Static files served directly (images, fonts, etc.)
- `web/src/`: Source files that need processing (Sass, TypeScript, etc.)
- `web/dist/`: Compiled assets (managed by build tools)

#### Production Deployment

##### Building for Production

To create a production-ready, self-contained binary with all assets embedded:

```sh
make build
```

This will:

1. Compile all Templ components to Go
2. Build and minify CSS/JS assets
3. Create a single binary at `./tmp/goby` with all assets embedded

The resulting binary includes all templates and static files using Go's `embed` package.

#### Systemd Service

For production deployments, you can use this systemd service file as a reference. Save it to `/etc/systemd/system/goby.service`:

```ini
[Unit]
Description=Goby Web Application
After=network.target

[Service]
Type=simple
User=www-data
Group=www-data
WorkingDirectory=/path/to/goby
ExecStart=/path/to/goby/goby
Restart=always
EnvironmentFile=/etc/goby/env

[Install]
WantedBy=multi-user.target
```

Create the environment file at `/etc/goby/env` with your production configuration. See the Configuration section below for all available options.

## Configuration

The application is configured using environment variables. For local development, you can create a `.env` file in the project root to manage these settings.

## Configuration Reference

| Variable             | Description                                                                           | Default                 | Required       |
| :------------------- | :------------------------------------------------------------------------------------ | :---------------------- | :------------- |
| **`SERVER_ADDR`**    | The address and port for the server to listen on.                                     | `:8080`                 | No             |
| **`APP_BASE_URL`**   | The public base URL for the application, used for generating links in emails.         | `http://localhost:8080` | No             |
| **`SESSION_SECRET`** | A long, random string used to secure user sessions.                                   | (none)                  | **Yes (Prod)** |
| **`APP_STATIC`**     | Controls static asset serving. `disk` for development, `embed` for production builds. | `disk`                  | No             |

### Database

| Variable           | Description                                     | Default                   | Required |
| :----------------- | :---------------------------------------------- | :------------------------ | :------- |
| **`SURREAL_URL`**  | The URL of your SurrealDB instance.             | `ws://localhost:8000/rpc` | **Yes**  |
| **`SURREAL_NS`**   | The namespace to use in SurrealDB.              | `app`                     | **Yes**  |
| **`SURREAL_DB`**   | The database to use in SurrealDB.               | `app`                     | **Yes**  |
| **`SURREAL_USER`** | The user for authenticating with SurrealDB.     | `app`                     | **Yes**  |
| **`SURREAL_PASS`** | The password for authenticating with SurrealDB. | `secret`                  | **Yes**  |

#### Email

| Variable             | Description                                                              | Default | Required                         |
| :------------------- | :----------------------------------------------------------------------- | :------ | :------------------------------- |
| **`EMAIL_PROVIDER`** | The email service to use (`log` or `resend`).                            | `log`   | No                               |
| **`EMAIL_API_KEY`**  | Your API key for the chosen email provider (e.g., Resend).               | (none)  | If `EMAIL_PROVIDER` is not `log` |
| **`EMAIL_SENDER`**   | The "from" address for outgoing emails (e.g., `noreply@yourdomain.com`). | (none)  | If `EMAIL_PROVIDER` is not `log` |

### Production Environment Example

Here is an example systemd service file for running the application in production.

```ini
[Service]
User=www-data
Group=www-data
Restart=always

Environment=SERVER_ADDR=:8080
Environment=APP_BASE_URL=https://yourdomain.com
Environment=SESSION_SECRET=a-very-long-and-random-secret-string
Environment=SURREAL_URL=ws://localhost:8000/rpc
Environment=SURREAL_NS=app
Environment=SURREAL_DB=app
Environment=SURREAL_USER=app
Environment=SURREAL_PASS=secret
Environment=EMAIL_PROVIDER=resend
Environment=EMAIL_API_KEY=your-resend-api-key
Environment=EMAIL_SENDER=noreply@yourdomain.com
ExecStart=/opt/goby/goby
WorkingDirectory=/opt/goby
```

#### Testing

This project includes both unit and integration tests to ensure code quality and correctness.

#### Running Tests

To run the entire test suite, which includes both unit and integration tests, use the following command:

```sh
make test
```

This command executes `go test ./...` with the appropriate build tags.

#### Test Database

Integration tests require a running test database, separate from your development database, to avoid data conflicts. Configuration for the test suite is managed in a `.env.test` file in the project root. To set it up, you can copy your existing `.env` file and modify the database connection details:

```sh
cp .env .env.test
```

#### Test Types

- **Unit Tests**: These tests focus on small, isolated pieces of code, like a single function or method. They do not require a database or other external services and are typically very fast.
- **Integration Tests**: These tests verify that different parts of the application work together correctly. For example, an integration test might check that an HTTP handler correctly interacts with the database. These tests are tagged with `//go:build integration` and require a live test database to run.
