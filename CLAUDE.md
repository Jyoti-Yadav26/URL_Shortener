# URL Shortener (Go)

A URL shortener built with Go, chi, pgx, Postgres, Redis, and Docker.

## Who this is for

The developer is an experienced Java/Spring Boot engineer who is new to Go.
Explanations should lean on Spring Boot comparisons where that shortens the
distance to understanding (e.g. "a Go interface satisfied implicitly is like
coding to a Spring `@Service` interface without `implements`, minus the
annotation").

## Working style

- Work in small phases. Stop after each phase for review before continuing
  to the next — do not chain multiple phases together unprompted.
- After writing code, explain:
  - Every new file: what it's for and why it exists where it does.
  - Every non-obvious Go idiom used in it, with a Spring Boot/Java
    comparison when that helps (e.g. zero values vs. null, `defer` vs.
    `finally`, implicit interface satisfaction vs. `implements`, struct
    embedding vs. inheritance).
- Don't add features, endpoints, config, or abstractions that weren't asked
  for. No speculative generalization.

## End of each phase

Once the phase works and `go test ./...` passes:

1. Show `git status` and a diff summary (`git diff --stat`).
2. Commit with a descriptive message.
3. Push to `origin main`.

Never commit `.env` files, credentials, keys, or any other secrets. If a
phase needs a secret, it goes in `.env` (gitignored) with a committed
`.env.example` holding placeholder values.

## Architecture

Layered, idiomatic Go:

- **handler** — HTTP layer (chi routes, request/response), thin.
- **service** — business logic.
- **repository** — data access (Postgres via pgx, Redis).

Rules:

- Interfaces are defined where they're *used* (consumer-side), not where
  they're implemented — e.g. the service package declares the repository
  interface it needs; the repository package just returns a concrete type
  that happens to satisfy it.
- Errors are returned, not panicked, and wrapped with `fmt.Errorf("...: %w", err)`
  to preserve the chain for `errors.Is`/`errors.As`.
- Every function doing I/O (DB, Redis, HTTP calls) takes `context.Context`
  as its first parameter.
- Table-driven tests for logic (`[]struct{ name string; ...; want ... }`
  with `t.Run` per case).

## Stack

- Router: chi
- Postgres driver: pgx
- Cache: Redis
- Containerization: Docker
