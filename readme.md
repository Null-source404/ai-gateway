# ai-gateway

A small, self-hosted **AI gateway written in Go**. Your apps send requests to *one* endpoint on your own server. The gateway checks who is calling, applies limits, forwards the request to an AI provider (Anthropic, an OpenAI-compatible API, a local Ollama model, or a built-in mock), and logs what happened.

> **Status: planning.** No code yet. This README is the plan: what it is, why, how we will use it, what we will build in what order, and what you need installed.

---

## 1. What it is

Without a gateway, every app you build talks to an AI provider directly:

```
chatty-bot ──────────► Anthropic API
somasmart  ──────────► Anthropic API     (each app holds the real API key)
kariba     ──────────► OpenAI API
```

With a gateway, the apps talk to *you*, and only the gateway holds provider keys:

```
chatty-bot ─┐                         ┌──► Anthropic
somasmart  ─┼──► ai-gateway (Go) ─────┼──► OpenAI-compatible
kariba     ─┘   auth · limits · logs  ├──► Ollama (local, free)
                                      └──► mock (no key, no cost)
```

## 2. How we will use it

- **Chatty WhatsApp bot:** instead of calling Anthropic directly, the bot calls `POST /v1/chat` on the gateway with its own gateway key. When your credits run out, the gateway can fall back to a local model or the mock instead of the bot crashing.
- **SomaSmart:** a future "explain this answer" feature calls the gateway, so limits and spend are controlled in one place, per app.
- **Learning:** it is a practical way to learn Go's real strengths: concurrency, `context`, HTTP servers, streaming, testing, and small static binaries.

## 3. Advantages (and honest limits)

| Advantage | Why it matters |
|---|---|
| **Real keys live in one place** | Apps get revocable gateway keys. A leaked bot key doesn't expose your provider account. |
| **Swap providers without touching apps** | Change one config line, not three codebases. |
| **Rate limits and spend control** | A bug or abuse in one app can't drain your credits. |
| **Fallback and retries** | Provider down or out of credit? Try the next one. |
| **One log and one set of metrics** | See who called what, how slow, how often it failed. |
| **Works with no credits** | The mock and Ollama providers let you build and test for free. |

Limits: it adds one extra network hop; it is a single process (no clustering); it is a learning project, not a hardened product. See *Security notes* below.

---

## 4. Scope, in milestones

Each milestone ends with something that runs and has tests. Stop after any of them and you still have a working tool.

| # | Milestone | You learn |
|---|---|---|
| **M1** | HTTP server with `GET /healthz` and `POST /v1/chat` forwarding to the **mock** provider | `net/http`, JSON, project layout, first test |
| **M2** | Real provider adapter behind a `Provider` interface (Anthropic and/or OpenAI-compatible/Ollama) | interfaces, HTTP clients, timeouts, `context` |
| **M3** | Gateway API keys (stored hashed), auth middleware | middleware, constant-time compare, hashing |
| **M4** | Per-key rate limiting (token bucket) | goroutine-safe state, `sync`, table-driven tests |
| **M5** | Streaming responses (Server-Sent Events) | `http.Flusher`, streaming, cancellation |
| **M6** | Retries and fallback between providers | error handling, backoff, circuit-breaker idea |
| **M7** | Structured logs, request IDs, simple `/metrics` | `log/slog`, observability |
| **M8** | Dockerfile, Makefile, GitHub Actions CI | multi-stage builds, CI for Go |

**Out of scope for now:** a web dashboard, billing, multi-user admin UI, response caching, clustering. Add later only if a real need shows up.

### API sketch (M1–M3)

```
GET  /healthz                      -> {"status":"ok"}
POST /v1/chat                      Authorization: Bearer <gateway-key>
     { "model": "default", "messages": [{"role":"user","content":"hi"}] }
  -> { "id": "...", "provider": "mock", "content": "...", "usage": {"input_tokens": 3, "output_tokens": 9} }
```

The gateway uses its **own simple request/response shape**. Each provider adapter translates to and from it, which is the whole point of the `Provider` interface.

---

## 5. Project structure

```
ai-gateway/
├── cmd/
│   └── gateway/
│       └── main.go              # wiring only: load config, build server, handle shutdown
├── internal/
│   ├── config/                  # read environment variables, validate, fail fast
│   ├── server/                  # routes, handlers, middleware chain
│   │   ├── server.go
│   │   ├── handlers.go
│   │   └── middleware.go        # request ID, logging, auth, rate limit, panic recovery
│   ├── auth/                    # gateway keys: generate, hash, verify, store
│   ├── ratelimit/               # token bucket per key
│   ├── provider/                # the Provider interface + adapters
│   │   ├── provider.go          # interface and shared types
│   │   ├── mock.go              # no network, no key, deterministic
│   │   ├── openai_compat.go     # OpenAI-style APIs and Ollama
│   │   ├── anthropic.go         # Anthropic Messages API
│   │   └── router.go            # choose a provider, retries, fallback (M6)
│   └── observability/           # slog setup, metrics counters
├── docs/
│   └── decisions.md             # short notes: why you chose X over Y (great for interviews)
├── .github/workflows/ci.yml     # go vet, go test -race, build
├── .env.example                 # documented settings, no real secrets
├── .gitignore                   # .env, bin/, *.db
├── Dockerfile                   # multi-stage, non-root, small final image
├── Makefile                     # run, test, lint, build, docker
├── go.mod
└── README.md
```

**Why `internal/`?** Go refuses to let other modules import it, so you can reorganise freely without breaking anyone. **Why a thin `main.go`?** So the real logic is testable without starting a process.

The core abstraction (the one thing to get right early):

```go
// internal/provider/provider.go  (sketch, not final code)
type Provider interface {
    Name() string
    Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
    // M5 adds: ChatStream(ctx, req) (<-chan Chunk, error)
}
```

---

## 6. Requirements

**You need**
- **Go 1.22 or newer** (`go version`). Needed for the improved `http.ServeMux` patterns like `mux.HandleFunc("POST /v1/chat", ...)`, so we can avoid a router dependency.
- **Git** and a GitHub account (for CI).
- **A terminal and VS Code** with the official Go extension (installs `gopls`, the formatter and the debugger).

**You do not need (important)**
- **No provider API key or credits to start.** M1 uses the mock; Ollama (optional) gives a free local model for M2.
- **No database for M1–M2.** Keys can live in a small file or in memory at first; add SQLite later if you want persistence.

**Optional**
- **Docker** for M8.
- **Ollama** with a small model, for a free real-model test.
- **`make`** (or just run the commands by hand).

**Dependencies:** the plan is **standard library only** for M1–M7: `net/http`, `encoding/json`, `log/slog`, `context`, `sync`, `crypto/sha256`, `crypto/subtle`. Fewer dependencies means more Go learned. If you decide to skip hand-writing the limiter, `golang.org/x/time/rate` is the one acceptable extra.

**Configuration (environment variables)**

| Variable | Purpose | Example |
|---|---|---|
| `GATEWAY_ADDR` | listen address | `:8080` |
| `GATEWAY_PROVIDER` | default provider | `mock` |
| `ANTHROPIC_API_KEY` | only if using Anthropic | (unset until you have credits) |
| `OPENAI_BASE_URL` / `OPENAI_API_KEY` | OpenAI-compatible or Ollama | `http://localhost:11434/v1` |
| `GATEWAY_RATE_PER_MIN` | default rate limit per key | `60` |

---

## 7. Security notes (decide these before coding)

- **Never commit provider keys.** `.env` is git-ignored; CI uses throwaway values.
- **Store gateway keys hashed** (SHA-256 is fine for high-entropy random keys); compare with `crypto/subtle`.
- **Do not log prompts or responses by default.** They can contain personal data. Log metadata (key id, provider, latency, token counts, status).
- **Set timeouts everywhere:** server read/write timeouts, client timeouts, and a `context` deadline per request. A hung upstream must not hang your server.
- **Limit request body size** (`http.MaxBytesReader`).
- **Bind to localhost** unless you deliberately expose it, and put it behind HTTPS (a reverse proxy) before any public use.

## 8. Definition of done (for the whole project)

- `make test` passes with the race detector on (`go test -race ./...`)
- Works end to end against the mock with zero credentials
- One real provider adapter tested (Ollama or Anthropic)
- A key can be revoked, and a rate-limited key gets `429` with `Retry-After`
- CI is green on GitHub, Docker image builds, README quick-start works from a clean clone
- `docs/decisions.md` has 5+ short entries you can explain out loud

## 9. Suggested first session (about 1 hour)

1. `mkdir ai-gateway && cd ai-gateway && go mod init github.com/<you>/ai-gateway`
2. Create the folders from section 5 (empty is fine) and the `.gitignore`.
3. **Write M1 yourself:** `/healthz`, then `/v1/chat` returning a fixed mock reply.
4. Write **one test** with `httptest` for `/healthz`.
5. Commit. Then ask your AI to review it, not write it.