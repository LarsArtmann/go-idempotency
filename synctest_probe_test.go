package idempotency_test

import (
	"context"
	"testing"
	"time"

	"testing/synctest"

	"github.com/larsartmann/go-idempotency"
)

func TestSynctestProbe(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		store := idempotency.NewMemoryStore(5 * time.Millisecond)

		ctx := context.Background()
		if err := store.CheckAndRecord(ctx, "probe", 20*time.Millisecond); err != nil {
			t.Errorf("CheckAndRecord: %v", err)

			return
		}

		time.Sleep(40 * time.Millisecond)

		seen, err := store.Seen(ctx, "probe")
		if err != nil {
			t.Errorf("Seen: %v", err)

			return
		}

		if seen {
			t.Error("claim still seen after 40ms virtual time (20ms TTL, 5ms sweep)")
		}

		store.Close()
		synctest.Wait()
	})
}
