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
		store := idempotency.NewMemoryStore(0)
		defer store.Close()

		ctx := context.Background()
		key := rapid.String().Draw(t, "key")

		if err := store.Record(ctx, key, idempotency.Forever); err != nil {
			t.Fatalf("Record(Forever): %v", err)
		}

		// A later short Record must be a no-op on the live permanent claim.
		if err := store.Record(ctx, key, 2*time.Millisecond); err != nil {
			t.Fatalf("Record(short) after Forever: %v", err)
		}

		time.Sleep(5 * time.Millisecond)

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
