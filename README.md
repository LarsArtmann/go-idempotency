# go-idempotency

A Go library providing an idempotency/deduplication store for CQRS command keys.

## Why

Delivery in a CQRS system is at-least-once: a client submits a command, loses the acknowledgement, and retries. Without deduplication, the retried command executes twice, producing duplicate events and side effects.

This library closes that gap. The client attaches a stable key to each logical command; the store records the key before processing and rejects retries whose key has already been recorded.

## Quick Start

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

## Features

- **Atomic `CheckAndRecord`** — single-lock check-and-set preventing the TOCTOU race
- **TTL-based expiration** — dual mechanism: background sweep goroutine + lazy deletion on read
- **Conflict-classified errors** — `ErrDuplicate` is an HTTP 409 Conflict via [go-error-family](https://github.com/larsartmann/go-error-family), non-retryable
- **Input validation** — non-positive TTL is rejected with `ErrInvalidTTL` (HTTP 400 Rejection), preventing silent exactly-once breakage
- **Concurrency-safe** — tested with 200 goroutines and property-based tests
- **Configurable sweep** — disable the background goroutine (`sweepInterval == 0`); lazy deletion still bounds memory growth

See [FEATURES.md](FEATURES.md) for the full inventory.

## Installation

```bash
go get github.com/larsartmann/go-idempotency
```

Requires Go 1.26+.

## Development

```bash
go test ./...          # run all tests
go test ./... -race    # run with race detector (mandatory)
go vet ./...           # static analysis
```

See [AGENTS.md](AGENTS.md) for architecture, conventions, and non-obvious design decisions.

## License

[MIT](LICENSE)
