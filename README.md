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
default_model = "gpt-5.2-codex"
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CODEX_ADDRESS` | `localhost:8080` | Server bind address |
| `CODEX_PATH` | `codex` | Path to codex CLI binary |
| `CODEX_TIMEOUT` | `30s` | Request timeout |
| `CODEX_JWT_SECRET` | (empty) | JWT secret (empty = no auth) |
| `CODEX_DEFAULT_MODEL` | `gpt-5.2-codex` | Default GPT model |

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

### Option 1: Ollama API (Simplest)

Many tools support Ollama out of the box:

```sh
# Start codex-serve
./codex-serve &

# Use with any Ollama-compatible tool
export OLLAMA_HOST=http://localhost:8080

# Example: if you have ollama CLI installed
ollama run codex "what is 2+2?"

# Or direct API call
curl -X POST http://localhost:8080/api/chat \
  -d '{"model":"codex","messages":[{"role":"user","content":"hello"}]}'
```

### Option 2: OpenAI API

For tools that use OpenAI format:

```sh
# Start codex-serve
./codex-serve &

# Configure tool to use custom endpoint
export OPENAI_BASE_URL=http://localhost:8080/v1
export OPENAI_API_KEY="dummy"  # Required by some tools, not validated

# Example with aider
aider --openai-api-base http://localhost:8080/v1
```

### Option 3: Pi Agent (Custom Provider)

Create `~/.pi/agent/models.json`:

```json
{
  "providers": {
    "codex-serve": {
      "baseUrl": "http://localhost:8080/v1",
      "api": "openai-completions",
      "apiKey": "dummy",
      "models": [
        {
          "id": "gpt-5.2-codex",
          "name": "GPT-5.2 Codex (Local)",
          "reasoning": true,
          "contextWindow": 200000,
          "maxTokens": 16000
        }
      ]
    }
  }
}
```

Then run: `pi "your prompt here"`

**Note**: Requires Node.js v20+ for pi agent.

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

### Available models

codex CLI supports OpenAI GPT models:
- `gpt-5.2-codex` (default) - Latest frontier agentic coding model
- `gpt-5.2` - Latest frontier model with improvements across knowledge, reasoning and coding
- `gpt-5.1-codex-max` - Codex-optimized flagship for deep and fast reasoning
- `gpt-5.1-codex-mini` - Optimized for codex. Cheaper, faster, but less capable

Access legacy models via: `codex -m <model_name>`

## License

MIT
