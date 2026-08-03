package idempotency_test

import (
	"context"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-idempotency"
)

// generateKeys pre-creates b.N unique string keys so that benchmarks measure
// store performance, not strconv formatting.
func generateKeys(n int) []string {
	keys := make([]string, n)
	for i := range n {
		keys[i] = strconv.Itoa(i)
	}

	return keys
}

// BenchmarkCheckAndRecord_Serial measures the cost of a single-goroutine
// CheckAndRecord call against a fresh key. No contention. Keys are
// pre-generated to isolate store performance from string formatting.
func BenchmarkCheckAndRecord_Serial(b *testing.B) {
	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	ctx := context.Background()
	keys := generateKeys(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	for i := range b.N {
		_ = store.CheckAndRecord(ctx, keys[i], time.Minute)
	}
}

// BenchmarkCheckAndRecord_Parallel_Contention measures CheckAndRecord under
// high contention: every goroutine hammers the same key. Only one goroutine
// wins; the rest get ErrDuplicate. This stresses the write lock.
func BenchmarkCheckAndRecord_Parallel_Contention(b *testing.B) {
	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	ctx := context.Background()
	_ = store.CheckAndRecord(ctx, "hot-key", time.Hour)

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = store.CheckAndRecord(ctx, "hot-key", time.Hour)
		}
	})
}

// BenchmarkCheckAndRecord_Parallel_UniqueKeys measures CheckAndRecord across
// many goroutines using unique keys (no contention, just lock overhead).
func BenchmarkCheckAndRecord_Parallel_UniqueKeys(b *testing.B) {
	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	ctx := context.Background()

	var counter atomic.Int64

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := counter.Add(1)
			_ = store.CheckAndRecord(ctx, strconv.FormatInt(i, 10), time.Minute)
		}
	})
}

// BenchmarkSeen_Hit measures Seen when the key exists and is not expired.
func BenchmarkSeen_Hit(b *testing.B) {
	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	ctx := context.Background()
	_ = store.Record(ctx, "seen-key", time.Hour)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		_, _ = store.Seen(ctx, "seen-key")
	}
}

// BenchmarkSeen_Miss measures Seen when the key does not exist.
func BenchmarkSeen_Miss(b *testing.B) {
	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		_, _ = store.Seen(ctx, "nonexistent")
	}
}

// BenchmarkRecord_NewKey measures Record against a fresh key. Keys are
// pre-generated to isolate store performance from string formatting.
func BenchmarkRecord_NewKey(b *testing.B) {
	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	ctx := context.Background()
	keys := generateKeys(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	for i := range b.N {
		_ = store.Record(ctx, keys[i], time.Minute)
	}
}
