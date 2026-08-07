package idempotency_test

import (
	"context"
	"runtime"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-idempotency"
)

// generateKeys pre-creates b.N unique string keys so that benchmarks measure
// store performance, not strconv formatting.
func generateKeys(n int) []string {
	keys := make([]string, n) //nolint:makezero // pre-allocated and filled by index
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

	for b.Loop() {
		_, _ = store.Seen(ctx, "seen-key")
	}
}

// BenchmarkSeen_Miss measures Seen when the key does not exist.
func BenchmarkSeen_Miss(b *testing.B) {
	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	ctx := context.Background()

	b.ReportAllocs()

	for b.Loop() {
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

// BenchmarkMemoryUsage_10KKeys measures the heap growth after recording 10,000
// unique keys. Reports bytes-per-key as a custom metric so the cost of each
// stored entry is visible for capacity planning.
func BenchmarkMemoryUsage_10KKeys(b *testing.B) {
	const keyCount = 10000

	ctx := context.Background()
	keys := generateKeys(keyCount)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		store := idempotency.NewMemoryStore(0)

		var before runtime.MemStats
		runtime.ReadMemStats(&before)

		for i := range keyCount {
			_ = store.Record(ctx, keys[i], time.Hour)
		}

		var after runtime.MemStats
		runtime.ReadMemStats(&after)

		store.Close()

		bytesPerKey := float64(after.HeapInuse-before.HeapInuse) / float64(keyCount)
		b.ReportMetric(bytesPerKey, "bytes/key")
	}
}

// BenchmarkMemoryUsage_AfterSweep measures how much memory the sweeper reclaims
// after all keys expire. Reports the percentage of heap reclaimed, verifying
// that the sweep goroutine actually frees memory and does not leak.
func BenchmarkMemoryUsage_AfterSweep(b *testing.B) {
	const keyCount = 10000

	ctx := context.Background()
	keys := generateKeys(keyCount)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		store := idempotency.NewMemoryStore(5 * time.Millisecond)

		for i := range keyCount {
			_ = store.Record(ctx, keys[i], 2*time.Millisecond)
		}

		var peak runtime.MemStats
		runtime.ReadMemStats(&peak)

		// Wait for the sweeper to run multiple cycles past the TTL.
		time.Sleep(30 * time.Millisecond)

		var after runtime.MemStats
		runtime.ReadMemStats(&after)

		store.Close()

		if peak.HeapInuse > 0 {
			reclaimed := float64(peak.HeapInuse-after.HeapInuse) / float64(peak.HeapInuse) * 100
			b.ReportMetric(reclaimed, "%-reclaimed")
		}
	}
}
