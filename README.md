# Goby

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

This will start both processes defined in the `Procfile`. Your application will be available at `http://localhost:3000` and will automatically reload when you make changes to Go or CSS files.

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

This setup achieves the same result, with your Go application running on `http://localhost:3000` and live-reloading enabled for both backend and frontend changes.
