# Goby

<p align="center">
  <img src="web/static/img/logo.svg" alt="Goby Mascot" width="200">
</p>

Goby is a project template for building web applications with Go and Tailwind CSS, featuring live-reloading for a great developer experience.

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
