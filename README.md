# go-idempotency

**Turns at-least-once delivery into at-most-once processing.**

[![CI](https://github.com/larsartmann/go-idempotency/actions/workflows/ci.yml/badge.svg)](https://github.com/larsartmann/go-idempotency/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-idempotency.svg)](https://pkg.go.dev/github.com/larsartmann/go-idempotency)
[![Go Report Card](https://goreportcard.com/badge/github.com/larsartmann/go-idempotency)](https://goreportcard.com/report/github.com/larsartmann/go-idempotency)
[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

A focused Go library that deduplicates retried operations using idempotency keys, so a retried command never executes twice. Built for CQRS command handling, it works for any at-most-once-processing need.

## Why

Delivery in a CQRS system is at-least-once: a client submits a command, loses the acknowledgement, and retries. Without deduplication, the retried command executes twice, producing duplicate events and side effects.

This library closes that gap. The client attaches a stable key to each logical command; the store records the key before processing and rejects retries whose key has already been recorded.

## How it works

```
1. client → submit (key=K)  →  CheckAndRecord(K)  →  key unseen   →  process  →  ack
2. ack lost; client retries →  CheckAndRecord(K)  →  ErrDuplicate  →  drop the retry
```

The entire correctness guarantee rests on `CheckAndRecord` being **atomic** — the check and the record happen under a single lock, so concurrent callers with the same key are serialized: exactly one wins.

```go
// ✅ Atomic — exactly one concurrent caller wins
err := store.CheckAndRecord(ctx, key, ttl)

// ❌ Racy — a TOCTOU window lets two callers both pass Seen before either records
seen, _ := store.Seen(ctx, key)
if !seen {
    _ = store.Record(ctx, key, ttl)
}
```

Prefer `CheckAndRecord`. Only reach for `Seen` / `Record` separately when you understand the race you are accepting.

## Quick start

```bash
go get github.com/larsartmann/go-idempotency
```

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/larsartmann/go-idempotency"
)

func main() {
	store := idempotency.NewMemoryStore(5 * time.Minute)
	defer store.Close()

	ctx := context.Background()
	cmdKey := "order-123-create"

	// Check-and-record in a single atomic step (preferred over Seen + Record).
	err := store.CheckAndRecord(ctx, cmdKey, 10*time.Minute)
	if errors.Is(err, idempotency.ErrDuplicate) {
		fmt.Println("already processed — dropping retry")
		return
	}
	if errors.Is(err, idempotency.ErrInvalidTTL) {
		fmt.Printf("programmer error: %v\n", err)
		return
	}
	if err != nil {
		fmt.Printf("store failure: %v\n", err)
		return
	}

	// Process the command...
	fmt.Println("processing command")
}
```

## Errors

Errors are classified by [go-error-family](https://github.com/larsartmann/go-error-family), so they map cleanly onto HTTP statuses and retry decisions downstream. Always check with `errors.Is`.

| Error           | Returned by                | Family    | HTTP | Retryable |
| --------------- | -------------------------- | --------- | ---- | --------- |
| `ErrDuplicate`  | `CheckAndRecord`           | Conflict  | 409  | No        |
| `ErrInvalidTTL` | `Record`, `CheckAndRecord` | Rejection | 400  | No        |

`ErrDuplicate` means a prior, still-valid recording exists — return it to the client, do not retry. `ErrInvalidTTL` means the caller passed `ttl <= 0`; a non-positive TTL records an already-past expiry that protects nothing, so the store rejects it loudly instead of silently breaking exactly-once.

## Features

- **Atomic `CheckAndRecord`** — single-lock check-and-set preventing the TOCTOU race
- **TTL-based expiration** — background sweep goroutine + lazy deletion on read
- **Conflict-classified errors** — `ErrDuplicate` is HTTP 409, non-retryable; `ErrInvalidTTL` is HTTP 400
- **Concurrency-safe** — verified with 200 goroutines and property-based tests
- **Configurable sweep** — disable the background goroutine (`sweepInterval == 0`); lazy deletion still bounds memory growth
- **Graceful shutdown** — `Close()` is idempotent and stops the sweeper

See [FEATURES.md](FEATURES.md) for the full, code-evidenced inventory.

## Status & roadmap

`MemoryStore` (single-process, in-memory) is stable and concurrency-tested today. The `Store` interface is designed for distributed backends — **Redis** (`SET NX`) and **SQL** (`INSERT ... ON CONFLICT DO NOTHING`) are planned but not yet implemented.

Versioning: **v0.x** — `MemoryStore` is stable, but the `Store` interface may gain methods (`Delete`, `Stats`) as backends land. **v1.0** ships when a persistent backend exists. See [ROADMAP.md](ROADMAP.md).

## Documentation

- [API reference (pkg.go.dev)](https://pkg.go.dev/github.com/larsartmann/go-idempotency)
- [Features](FEATURES.md) — honest feature inventory with code evidence
- [Domain language](docs/DOMAIN_LANGUAGE.md) — glossary of idempotency terms
- [Changelog](CHANGELOG.md)
- [Contributing](CONTRIBUTING.md)

## Development

```bash
go test ./... -race    # tests with race detector (mandatory)
go vet ./...           # static analysis
golangci-lint run ./... # lint (60+ linters, see .golangci.yml)
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for the full setup, testing strategy, and conventions.

## License

[MIT](LICENSE)
