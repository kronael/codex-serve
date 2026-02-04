# codex-serve

HTTP server wrapping OpenAI Codex CLI with Ollama, OpenAI, and streaming APIs.

Skills: /go, /cli, /service

## Tasks

### Phase 1: Foundation

- Create go.mod with module codex-serve, require gorilla/websocket, golang-jwt/jwt/v5, BurntSushi/toml, google/uuid, prometheus/client_golang

- Create models.go with structs: Message (Role, Content), OllamaChatRequest, OllamaChatResponse, OpenAIChatRequest, OpenAIChatResponse, AnthropicMessageRequest, AnthropicMessageResponse

- Create errors.go with APIError struct and error code constants (ErrInvalidRequest, ErrUnauthorized, ErrTimeout, ErrCodexFailed), WriteError helper

- Create config.go with Config struct and LoadConfig that reads TOML then env vars (CODEX_ADDRESS, CODEX_PATH, CODEX_TIMEOUT, CODEX_JWT_SECRET)

### Phase 2: Core

- Create codex.go with CodexClient that spawns "codex exec --quiet --output-format json", streams stdout, handles SIGTERM/SIGKILL on cancel

- Create metrics.go with Prometheus metrics: requests_total counter, request_duration histogram, active_sessions gauge, tokens_total counter

- Create auth.go with JWT middleware, GenerateSecret and GenerateToken functions

- Create session.go with Session struct (ID, History, Usage), SessionManager with Create/Get/Delete/List methods, cleanup goroutine

### Phase 3: Handlers

- Create ollama.go with /api/tags GET, /api/chat POST (NDJSON stream), /api/generate POST handlers

- Create openai.go with /v1/models GET, /v1/chat/completions POST (SSE stream) handlers

- Create anthropic.go with /v1/messages POST handler using Anthropic SSE event format

- Create websocket.go with /ws/session handler for interactive sessions with resume support

### Phase 4: Server

- Create server.go with NewServer that mounts all routes, applies auth middleware, returns http.Server with graceful shutdown

- Create main.go with CLI (config file arg, jwt subcommand), starts server, handles SIGINT/SIGTERM

- Create Makefile with build, test, smoke, lint, clean targets
