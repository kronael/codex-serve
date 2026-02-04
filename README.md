# codex-serve

HTTP server exposing codex via Ollama, OpenAI, and Anthropic
compatible APIs.

## Requirements

- Go 1.23+
- [codex CLI](https://docs.anthropic.com/en/docs/claude-code) installed
  and authenticated

## Quick Start

First, authenticate with codex:

```sh
claude auth login
```

Then build and run codex-serve:

```sh
make build
./codex-serve
```

Server runs at http://localhost:8080

Test:
```sh
curl http://localhost:8080/health
curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{"model":"codex","messages":[{"role":"user","content":"hello"}]}'
```

## Development

```sh
make build   # Build binary
make test    # Run unit tests (<5s)
make smoke   # Run integration tests
make lint    # Run linter
make run     # Build and run server
make clean   # Clean build artifacts
```

## Configuration

TOML config file or environment variables.

Config precedence (higher overrides lower):
1. Environment variables
2. `./.codex-serve` (local)
3. `./.claude-serve` (local, fallback for backwards compatibility)
4. `~/.codex-serve/config` (global)
5. `~/.claude-serve/config` (global, fallback)

### Config File

```toml
address = "localhost:8080"
path = "codex"
timeout = "30s"
jwt_secret = ""
default_model = "claude-3-5-sonnet-20241022"
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CODEX_ADDRESS` | `localhost:8080` | Server bind address |
| `CODEX_PATH` | `codex` | Path to codex CLI binary |
| `CODEX_TIMEOUT` | `30s` | Request timeout |
| `CODEX_JWT_SECRET` | (empty) | JWT secret (empty = no auth) |
| `CODEX_DEFAULT_MODEL` | `claude-3-5-sonnet-20241022` | Default Claude model |

## Authentication

```sh
./codex-serve -jwt secret           # Generate secret
./codex-serve -jwt token <secret>   # Generate token
```

When `CODEX_JWT_SECRET` is set:
- All endpoints require `Authorization: Bearer <token>`

## API

### Health

```sh
curl http://localhost:8080/health
# {"status":"ok","codex":"codex"}
```

### Metrics

```sh
curl http://localhost:8080/metrics
```

### Ollama API (`/api/*`)

```sh
# List models
curl http://localhost:8080/api/tags

# Chat (streaming NDJSON)
curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{"model":"codex","messages":[{"role":"user","content":"hello"}],"stream":true}'

# Generate
curl -X POST http://localhost:8080/api/generate \
  -H "Content-Type: application/json" \
  -d '{"model":"codex","prompt":"hello","stream":true}'
```

### OpenAI API (`/v1/*`)

```sh
# List models
curl http://localhost:8080/v1/models

# Chat completions (streaming SSE)
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"codex","messages":[{"role":"user","content":"hello"}],"stream":true}'
```

### Anthropic API (`/v1/messages`)

```sh
curl -X POST http://localhost:8080/v1/messages \
  -H "Content-Type: application/json" \
  -d '{"model":"codex","messages":[{"role":"user","content":"hello"}],"max_tokens":1024,"stream":true}'
```

### WebSocket Sessions (`/ws/session`)

Interactive sessions with resume support.

Messages:
- `{"type":"create"}` - Create new session
- `{"type":"resume","session_id":"..."}` - Resume session
- `{"type":"prompt","content":{"prompt":"..."}}` - Send prompt
- `{"type":"close"}` - Close session

### Sessions

```sh
curl http://localhost:8080/v1/sessions
# {"sessions":["uuid-1","uuid-2"],"count":2}
```

## Usage with AI Agents

Use codex-serve as backend for OpenAI-compatible agents:

```sh
# Start codex-serve
./codex-serve &

# Point your agent at it
export OPENAI_BASE_URL=http://localhost:8080/v1

# Run pi agent with codex backend
pi "refactor this module to use dependency injection"

# Or any OpenAI-compatible tool
aider --openai-api-base http://localhost:8080/v1
```

Swap backends without changing agent code.

## Troubleshooting

### codex CLI not found

If you get "exec: codex: executable file not found", the codex CLI is not in your PATH.

Options:
1. Install codex CLI globally (recommended)
2. Set `CODEX_PATH` to the full path: `export CODEX_PATH=/path/to/codex`
3. Set `path = "/path/to/codex"` in config file

Check codex is installed:
```sh
codex --version
```

### Authentication errors

codex-serve requires the codex CLI to be authenticated:

```sh
claude auth login
```

After authentication, verify with:
```sh
codex exec "what is 2+2?"
```

### Config file not found

Config files are optional. If not found, codex-serve uses defaults.

To verify config is loaded, check startup logs or test with:
```sh
CODEX_ADDRESS=127.0.0.1:9999 ./codex-serve
```

### Streaming output format

codex CLI outputs JSON-line format (`--json` flag):
```json
{"type":"content","content":{"text":"Hello"}}
{"type":"done"}
```

codex-serve transforms this into the appropriate API format (SSE/NDJSON).

## License

MIT
