# Troubleshooting Guide

Common issues and solutions when working with Goby.

## Installation Issues

### Overmind not found

**Problem**: `make dev` fails with "overmind: command not found"

**Solution**:
```sh
# Install with Go
go install github.com/DarthSim/overmind/v2@latest

# Or on macOS
brew install overmind
```

Alternatively, run processes manually:
```sh
# Terminal 1
air

# Terminal 2
templ generate --watch

# Terminal 3
npm run dev:tailwind
```

### Node modules not installing

**Problem**: `npm install` fails

**Solution**:
```sh
# Clear cache and retry
rm -rf node_modules package-lock.json
npm cache clean --force
npm install
```

## Database Issues

### Cannot connect to SurrealDB

**Problem**: Application fails to start with database connection errors

**Solution**:
1. Ensure SurrealDB is running (choose one method):
   
   **Docker:**
   ```sh
   docker run --rm -p 8000:8000 surrealdb/surrealdb:latest \
     start --log trace --user root --pass root
   ```
   
   **Native binary:**
   ```sh
   surreal start --log trace --user root --pass root
   ```
   
   **Or use your existing SurrealDB instance**

2. Verify `.env` configuration matches:
   ```env
   SURREAL_URL=ws://localhost:8000
   SURREAL_USER=root
   SURREAL_PASS=root
   ```

3. Check if port 8000 is already in use:
   ```sh
   lsof -i :8000
   ```

### Database timeout errors

**Problem**: Queries timeout or fail intermittently

**Solution**:
Increase timeout values in `.env`:
```env
DB_QUERY_TIMEOUT=10s
DB_EXECUTE_TIMEOUT=15s
```

## Build Issues

### Templ files not compiling

**Problem**: Changes to `.templ` files don't appear

**Solution**:
1. Ensure templ is installed:
   ```sh
   go install github.com/a-h/templ/cmd/templ@latest
   ```

2. Manually regenerate:
   ```sh
   templ generate
   ```

3. Check for syntax errors in `.templ` files

### CSS not updating

**Problem**: Tailwind CSS changes don't appear

**Solution**:
1. Ensure Tailwind is watching:
   ```sh
   npm run dev:tailwind
   ```

2. Clear the generated CSS:
   ```sh
   rm web/static/css/style.css
   ```

3. Restart the Tailwind process

## Runtime Issues

### Port already in use

**Problem**: Server won't start - port 8080 in use

**Solution**:
1. Find and kill the process:
   ```sh
   lsof -i :8080
   kill -9 <PID>
   ```

2. Or change the port in `.env`:
   ```env
   SERVER_ADDR=:3000
   ```

### Session errors

**Problem**: "Invalid session" or authentication issues

**Solution**:
1. Ensure `SESSION_SECRET` is set in `.env`
2. Clear browser cookies for localhost
3. Restart the server

### WebSocket connection fails

**Problem**: Real-time features don't work

**Solution**:
1. Check browser console for WebSocket errors
2. Verify WebSocket endpoints are registered
3. Ensure pub/sub system is running
4. Check that topics are properly registered

## Module Development Issues

### Module not loading

**Problem**: New module doesn't appear in the application

**Solution**:
1. Verify module is registered in `internal/app/modules.go`
2. Check that dependencies are added to `internal/app/dependencies.go`
3. Ensure module's `Boot()` method doesn't return an error
4. Check logs for initialization errors

### Routes not working

**Problem**: Module routes return 404

**Solution**:
1. Verify routes are registered in module's `Boot()` method
2. Check route path - module routes are under `/app/<modulename>/`
3. Ensure Echo group is being used correctly
4. Check for middleware that might be blocking requests

## CLI Tool Issues

### goby-cli command not found

**Problem**: `./goby-cli` or `goby-cli` not found

**Solution**:
```sh
# Build the CLI
go build -o goby-cli ./cmd/goby-cli

# Or install to PATH
make install-cli
```

### Module generation fails

**Problem**: `goby-cli new-module` fails

**Solution**:
1. Ensure you're in the project root directory
2. Check that `internal/app/modules.go` exists
3. Verify Go syntax in existing files
4. Try with `--minimal` flag for simpler module

## Performance Issues

### Slow hot-reload

**Problem**: Changes take a long time to reflect

**Solution**:
1. Reduce number of watched files in `.air.toml`
2. Exclude large directories from file watchers
3. Use `--minimal` modules to reduce complexity
4. Consider increasing system file watch limits

### High memory usage

**Problem**: Application uses excessive memory

**Solution**:
1. Check for goroutine leaks in modules
2. Verify database connections are properly closed
3. Review pub/sub subscriptions for cleanup
4. Use profiling tools:
   ```sh
   go tool pprof http://localhost:8080/debug/pprof/heap
   ```

## Getting Help

If you're still stuck:

1. Check the [main README](../README.md) for setup instructions
2. Review [example modules](../internal/modules/examples/) for patterns
3. Check the [CLI documentation](../cmd/goby-cli/README.md)
4. Search existing GitHub issues
5. Create a new issue with:
   - Go version (`go version`)
   - Node version (`node --version`)
   - Operating system
   - Error messages and logs
   - Steps to reproduce
