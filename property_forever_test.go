package idempotency_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-idempotency"
	"pgregory.net/rapid"
)

// TestProperty_ForeverClaimNeverExpires: a Forever claim survives a later
// short-TTL Record (no downgrade) and remains a duplicate for check-and-record.
func TestProperty_ForeverClaimNeverExpires(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		clock := newFakeClock(time.Unix(0, 0))
		store := idempotency.NewMemoryStore(0, idempotency.WithClock(clock.Now))
		defer store.Close()

		ctx := context.Background()
		key := rapid.String().Draw(t, "key")

		if err := store.Record(ctx, key, idempotency.Forever); err != nil {
			t.Fatalf("Record(Forever): %v", err)
		}

		// A later short Record must be a no-op on the live permanent claim.
		// The short TTL is drawn at second scale and then fully elapses on the
		// pinned clock: if the claim had been downgraded, the key would read
		// unseen below.
		shortTTL := time.Duration(rapid.IntRange(1, 3600).Draw(t, "short_ttl_seconds")) * time.Second

		if err := store.Record(ctx, key, shortTTL); err != nil {
			t.Fatalf("Record(short) after Forever: %v", err)
		}

		clock.Advance(shortTTL)

		seen, err := store.Seen(ctx, key)
		if err != nil || !seen {
			t.Fatalf("Forever claim expired after a short Record: seen=%v err=%v", seen, err)
		}

		err = store.CheckAndRecord(ctx, key, idempotency.Forever)
		if !errors.Is(err, idempotency.ErrDuplicate) {
			t.Fatalf("CheckAndRecord on a Forever claim: want ErrDuplicate, got %v", err)
		}
	})
}
