# AI Gateway

This is a small Go HTTP service that gives applications one local endpoint for chat
requests. It checks a shared gateway bearer key, applies an in-memory request
limit, and sends the request to either a built-in mock or an
OpenAI-compatible provider such as Ollama.

This is a personal learning/prototype project, not a production-ready gateway.
It currently uses one shared API key and a process-local limiter; it does not
provide user management, streaming, billing, or durable metrics.

## What it does

The gateway accepts a small chat request format and returns its own response
format. Provider-specific request/response translation lives behind the
`provider.Provider` interface.

Current behavior:

- `GET /healthz` returns a basic health response.
- `POST /v1/chat` checks `Authorization: Bearer <gateway-key>`, then forwards
  the chat request to the selected provider.
- `GATEWAY_PROVIDER=mock` uses a deterministic mock and needs no network or AI
  credentials.
- `GATEWAY_PROVIDER=openai` or `ollama` uses the OpenAI-compatible adapter
  configured by `OPENAI_BASE_URL`, `OPENAI_API_KEY`, and `GATEWAY_MODEL`.
- If the configured provider fails, the router tries the mock provider.
- Request logs currently include the HTTP method, path, and elapsed time.

## Requirements

- Go 1.22 or newer
- PowerShell on Windows for the demo commands below (or adapt them for your
  shell)

There are no third-party Go module dependencies in the current `go.mod`.

## Run it locally

Open a terminal in the project folder. These commands start the server on
loopback only and use the mock provider, so they do not call an external AI
service or need an AI API key:

```powershell
$env:GATEWAY_ADDR = "127.0.0.1:8080"
$env:GATEWAY_PROVIDER = "mock"
$env:GATEWAY_API_KEY = "local-demo-key"
$env:GATEWAY_RATE_PER_MIN = "60"
go run ./cmd/gateway
```

Keep that terminal open. In a **second** PowerShell terminal, check health:

```powershell
Invoke-RestMethod -Uri "http://127.0.0.1:8080/healthz" -Method Get
```

Then send a chat request:

```powershell
$payload = @{
    model = "default"
    messages = @(
        @{ role = "user"; content = "Hello from my app" }
    )
} | ConvertTo-Json -Depth 5

Invoke-RestMethod `
    -Uri "http://127.0.0.1:8080/v1/chat" `
    -Method Post `
    -Headers @{ Authorization = "Bearer local-demo-key" } `
    -ContentType "application/json" `
    -Body $payload
```

The chat response should include a mock reply such as `Mock reply: Hello from
my app`. The health route does not require the gateway key; the chat route does.
Stop the server with **Ctrl+C** in its terminal.

If you prefer `.env` configuration, copy `.env.example` to `.env` only if you
do not already have a local `.env`, then edit the values. The application loads
`.env` from its current working directory, and environment variables already
set in the process take precedence. `.env` is ignored by Git.

## Configuration

| Variable | Default | Description |
|---|---|---|
| `GATEWAY_ADDR` | `:8080` | HTTP listen address. Use `127.0.0.1:8080` for local-only access. |
| `GATEWAY_PROVIDER` | `mock` | `mock`, `openai`, or `ollama`. The latter two use the same OpenAI-compatible adapter. |
| `GATEWAY_API_KEY` | `dev-gateway-key-change-me` | Shared bearer token required by `POST /v1/chat`. Set a private value for local use. An empty value disables this check. |
| `GATEWAY_RATE_PER_MIN` | `60` | Configures the in-memory limiter's capacity. See the prototype limitations below. |
| `GATEWAY_MODEL` | `default` | Model name passed to the provider when the request omits `model`. |
| `OPENAI_BASE_URL` | `http://localhost:11434/v1` | OpenAI-compatible API base URL; this is the usual local Ollama URL. |
| `OPENAI_API_KEY` | empty | Optional upstream bearer key; leave empty for a local Ollama instance if appropriate. |

For Ollama, start Ollama and make sure the configured model is available there,
then set `GATEWAY_PROVIDER=ollama` and `GATEWAY_MODEL` to that model's name.
For another OpenAI-compatible service, set its base URL and upstream key as
needed. Keep upstream credentials in environment variables or an untracked
`.env` file, never in source control.

## API

### Health

```http
GET /healthz
```

Example response:

```json
{"status":"ok"}
```

### Chat

```http
POST /v1/chat
Authorization: Bearer local-demo-key
Content-Type: application/json
```

Request body:

```json
{
  "model": "default",
  "messages": [
    {"role": "user", "content": "Hello"}
  ]
}
```

Response shape:

```json
{
  "id": "mock-1",
  "provider": "mock",
  "content": "Mock reply: Hello",
  "usage": {
    "input_tokens": 3,
    "output_tokens": 9
  }
}
```

The `usage` values in mock responses are fixed demo values, not measured token
counts.

## Test and build

From the project folder:

```powershell
go test ./...
go vet ./...
go build ./cmd/gateway
```

The existing tests cover the health and chat handlers (including gateway-key
checking and mock responses) and the OpenAI-compatible adapter's request
translation and response parsing. They use Go's `httptest` package, so they do
not need a running gateway or real provider credentials.

## Startup flow and layout

When you run `go run ./cmd/gateway`, `main` loads `.env` and process
configuration, validates it, chooses a provider, builds the HTTP routes and
middleware, then starts listening on `GATEWAY_ADDR`.

```text
cmd/gateway/main.go             startup and dependency wiring
internal/config/config.go       .env and environment-variable configuration
internal/provider/              provider interface, mock, adapter, fallback router
internal/server/                routes, HTTP handlers, auth, logging, rate limit
```

## Prototype limitations and safety

- The gateway key is one shared value, not separate keys per application; there
  is no key issuance, hashing/storage, or revocation flow.
- The limiter is in memory and shared by all requests in the process. Its
  current implementation refills its full capacity after roughly one second,
  so `GATEWAY_RATE_PER_MIN` should **not** be treated as a reliable per-minute
  or per-client quota.
- The mock fallback can make an upstream outage look like a successful chat
  response. Check the returned `provider` field when evaluating a real provider.
- The server does not yet configure production-grade HTTP server timeouts,
  request-body limits, TLS, or graceful shutdown.
- Logging is basic and there is no metrics endpoint or streaming API.

Keep it bound to loopback for local experiments. Do not expose it to the
internet or put real user traffic through it without adding appropriate
authentication, rate limiting, timeouts, input limits, and HTTPS.
