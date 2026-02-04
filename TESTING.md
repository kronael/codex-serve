# Testing Guide

## Running Tests

### Unit Tests

Fast tests that complete in <5s:

```sh
make test
```

Or directly:
```sh
go test -v -short -timeout 5s ./...
```

### Smoke Tests (Integration with Real Codex)

Smoke tests use the **real codex CLI** to verify the integration works correctly.

Requirements:
- codex CLI installed (`which codex`)
- codex authenticated (`claude auth login`)

Run smoke tests:
```sh
make smoke
```

Or directly:
```sh
go test -v -timeout 80s -run "Real|Smoke" ./...
```

These tests will skip if codex is not available.

### Specific Test

```sh
go test -v -run TestSessionLifecycle
```

## Test Structure

### Unit Tests

Unit tests are colocated with source files:

- `config_test.go` - Config loading and precedence
- `codex_test.go` - Codex CLI execution and streaming
- `session_test.go` - Session management and cleanup
- `ollama_test.go` - Ollama API handlers
- `openai_test.go` - OpenAI API handlers
- `anthropic_test.go` - Anthropic API handlers

### Integration Tests

Integration tests in `tests/` directory test full request flows.

## Testing Strategy

### Unit Tests (Fast, Mocked)

Unit tests use mock commands (like `echo`) instead of real codex:

```go
// Use echo to return fake JSON responses
client := NewCodexClient("echo", 5*time.Second)
```

This allows fast testing (~<1s) without requiring codex installation.

### Smoke Tests (Slower, Real)

Smoke tests in `integration_test.go` use the **real codex CLI** to verify:
- The mock data format matches real codex output
- The actual integration works end-to-end
- Parsing handles real codex responses correctly

This is critical because unit tests alone can't catch format mismatches between mocked and real data.

## Test Coverage

Run with coverage:

```sh
go test -cover ./...
```

Generate coverage report:

```sh
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Testing with Real codex CLI

To test with actual codex CLI:

1. Ensure codex is authenticated: `claude auth login`
2. Set path in tests or use default
3. Run integration tests

Example manual test:

```sh
# Start server
./codex-serve &
PID=$!

# Test health
curl http://localhost:8080/health

# Test chat
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"codex","messages":[{"role":"user","content":"what is 2+2?"}],"stream":false}'

# Cleanup
kill $PID
```

## Continuous Integration

Tests are designed to run in CI without external dependencies:
- Mock codex CLI with echo/sleep
- Use temp directories for config
- Short timeouts for fast feedback

## Writing New Tests

### Test Case Description

Add description comment at top of test file:

```go
// Test codex client handles streaming responses correctly
```

### Unit Test Template

```go
func TestFeatureName_Scenario(t *testing.T) {
	// Setup
	client := NewCodexClient("echo", 5*time.Second)

	// Execute
	result, err := client.DoSomething()

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result != expected {
		t.Errorf("expected %v, got %v", expected, result)
	}
}
```

### Integration Test Template

```go
func TestEndToEnd_Feature(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Setup server
	cfg := &Config{...}
	srv, _ := NewServer(cfg)
	go srv.Start()
	defer srv.Shutdown(context.Background())

	// Test full flow
	resp, err := http.Post(...)

	// Assert
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}
```

## Common Issues

### Test Timeout

If tests timeout, increase timeout:
```sh
go test -timeout 120s ./...
```

### Race Conditions

Run with race detector:
```sh
go test -race ./...
```

### Sandbox Restrictions

Tests must respect filesystem sandbox. Use temp directories from `t.TempDir()`.
