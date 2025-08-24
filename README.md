# Go Template Project

A Go project template with database connectivity and user management.

## Features

- Database connection management
- User management (CRUD operations)
- Environment-based configuration

## Prerequisites

- Go 1.21 or later
- SurrealDB (or compatible database)

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/go-template.git
   cd go-template
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Set up environment variables by creating a `.env` file:
   ```bash
   cp .env.example .env
   ```

4. Update the `.env` file with your database configuration.

## Configuration

The following environment variables are required:

```
SURREAL_URL=ws://localhost:8000/rpc
SURREAL_NS=test
SURREAL_DB=test
SURREAL_USER=root
SURREAL_PASS=root
```

## Usage

Run the application:
```bash
go run main.go
```

## Project Structure

```
go-template/
├── internal/
│   ├── config/      # Configuration management
│   └── database/    # Database connection and operations
├── .env.example     # Example environment variables
├── go.mod          # Go module definition
└── main.go         # Application entry point
```
