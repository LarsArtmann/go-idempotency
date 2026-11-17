package idemtest

import (
	"context"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-idempotency"
)

// Op names the recorded [idempotency.Store] operation.
type Op string

// The three Store operations, as they appear in [Call.Op].
const (
	OpSeen           Op = "Seen"
	OpRecord         Op = "Record"
	OpCheckAndRecord Op = "CheckAndRecord"
)

// Call is one observed store operation: which method ran, with which key and
// TTL, and what it returned.
type Call struct {
	Op  Op
	Key string
	TTL time.Duration // zero for Seen, which takes no TTL
	Err error         // the error the wrapped store returned
}

// Recorder is a spy around a [idempotency.Store]: every call is delegated to
// Next and appended to the record. Spying catches wiring bugs as assertion
// failures instead of silences — a pipeline that derives its key from
// per-request values, or claims with the wrong TTL, shows up in
// [Recorder.Calls] immediately.
type Recorder struct {
	Next idempotency.Store

	mu    sync.Mutex
	calls []Call
}

var _ idempotency.Store = (*Recorder)(nil)

// NewRecorder wraps next so every store call is recorded, then delegated.
func NewRecorder(next idempotency.Store) *Recorder {
	return &Recorder{Next: next} //nolint:exhaustruct_v5 // calls starts empty
}

// Seen delegates to Next and records the call.
func (r *Recorder) Seen(ctx context.Context, key string) (bool, error) {
	seen, err := r.Next.Seen(ctx, key)

	r.record(Call{Op: OpSeen, Key: key, TTL: 0, Err: err})

	return seen, err //nolint:wrapcheck // transparent delegation is the double's contract
}

// Record delegates to Next and records the call.
func (r *Recorder) Record(ctx context.Context, key string, ttl time.Duration) error {
	err := r.Next.Record(ctx, key, ttl)

	r.record(Call{Op: OpRecord, Key: key, TTL: ttl, Err: err})

	return err //nolint:wrapcheck // transparent delegation is the double's contract
}

// CheckAndRecord delegates to Next and records the call.
func (r *Recorder) CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error {
	err := r.Next.CheckAndRecord(ctx, key, ttl)

	r.record(Call{Op: OpCheckAndRecord, Key: key, TTL: ttl, Err: err})

	return err //nolint:wrapcheck // transparent delegation is the double's contract
}

// Calls returns a copy of the recorded calls, in call order.
func (r *Recorder) Calls() []Call {
	r.mu.Lock()
	defer r.mu.Unlock()

	return slices.Clone(r.calls)
}

// RequireClaimed fails tb unless the spied pipeline issued at least one
// CheckAndRecord for key with exactly ttl — the assertion that the idempotency
// wiring actually runs, with the key and window the caller configured.
func RequireClaimed(tb testing.TB, recorder *Recorder, key string, ttl time.Duration) {
	tb.Helper()

	for _, call := range recorder.Calls() {
		if call.Op == OpCheckAndRecord && call.Key == key && call.TTL == ttl {
			return
		}
	}

	tb.Fatalf(
		"no CheckAndRecord(key=%q, ttl=%v) recorded; calls: %s",
		key, ttl, formatCalls(recorder.Calls()),
	)
}

func (r *Recorder) record(call Call) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.calls = append(r.calls, call)
}

func formatCalls(calls []Call) string {
	if len(calls) == 0 {
		return "(none — the pipeline never touched the store)"
	}

	parts := make([]string, 0, len(calls))
	for _, call := range calls {
		parts = append(parts, string(call.Op)+"("+call.Key+", "+call.TTL.String()+")")
	}

	return strings.Join(parts, ", ")
}
