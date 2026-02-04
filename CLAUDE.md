# codex-serve Architecture

## Project Layout

```
codex-serve/
├── main.go        # CLI entry point, signal handling
├── config.go      # Config loading (TOML, env vars)
├── server.go      # HTTP server, route mounting
├── codex.go       # Claude CLI subprocess client
├── auth.go        # JWT middleware and token generation
├── session.go     # Session state management
├── metrics.go     # Prometheus metrics
├── models.go      # Request/response structs
├── errors.go      # Error types and helpers
├── ollama.go      # Ollama API handlers
├── openai.go      # OpenAI API handlers
├── anthropic.go   # Anthropic API handlers
└── websocket.go   # WebSocket session handler
```

## Patterns

### Process Management

Graceful shutdown: SIGTERM, wait 5s, then SIGKILL.

```go
proc.Signal(syscall.SIGTERM)
timer := time.NewTimer(5 * time.Second)
select {
case <-done:
    timer.Stop()
case <-timer.C:
    proc.Kill()
}
```

### Config Loading

Precedence (higher overrides lower):
1. Environment variables (CODEX_*)
2. `./.claude-serve` (local TOML)
3. `~/.claude-serve/config` (global TOML)

### Error Handling

Never silent failures. All errors surface with context:

```go
return fmt.Errorf("failed to start codex: %w", err)
```

### State Management

Use sync.RWMutex for concurrent access:

```go
s.mu.Lock()
defer s.mu.Unlock()
s.State = newState
```

## Session States

```
CREATED -> RUNNING -> COMPLETED
                  \-> FAILED
           \-> CLOSED (cancelled)
```

State transitions are atomic via sync.Mutex.

## Testing

```bash
make test   # Unit tests (<5s)
make smoke  # Integration tests
```

Mock subprocess calls, not real Claude CLI.

## Development Commands

```bash
make build  # Build binary to ./dist/codex-serve
make test   # Run unit tests
make lint   # Run go vet and go fmt
make run    # Build and run server
make clean  # Clean build artifacts
```
