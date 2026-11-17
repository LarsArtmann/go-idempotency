// Package idemtest provides domain-driven test helpers for code that uses
// go-idempotency — the drills and doubles that every consumer otherwise
// hand-rolls (ADR-007).
//
// It solves the three problems that make idempotency integrations hard to
// test:
//
//  1. You need a [idempotency.Store] in tests, but the production backend is
//     Redis or SQL and the deprecated root in-memory store is not a blessed
//     answer. [NewStore] is: a minimal, contract-certified test double.
//  2. You must prove at-most-once behavior through YOUR wiring — a retry
//     storm, a concurrent race, a failing store. The [RequireExactlyOnce],
//     [RequireOneWinner], and [RequireFailClosed] drills drive your dispatch
//     function and assert the outcome.
//  3. You must drive the HTTP adapter end-to-end. [Send] issues a request
//     with an Idempotency-Key and the Require* assertions name the adapter's
//     contract (duplicate, missing key, store unavailable).
//
// Import idemtest from _test.go files only, and it never enters a production
// import path — the httptest.Server precedent.
//
// # Quick start: a store and a drill
//
//	func TestPlaceOrder(t *testing.T) {
//	    t.Parallel()
//
//	    store := idemtest.NewStore()
//	    exec := idemtest.NewExecutions()
//	    place := middleware.NewCommand(store, time.Minute, exec.Fn())
//
//	    idemtest.RequireExactlyOnce(t, place, "order-42:place", 5, exec)
//	}
//
// The dispatch drills take any [Dispatch] — a [middleware.Command] satisfies
// it without conversion, and so does a closure over
// [idempotency.Store.CheckAndRecord] when the pipeline does not use the
// middleware package.
//
// # Concurrency: the TOCTOU detector
//
// [RequireOneWinner] releases N goroutines on one key at once and fails
// unless exactly one executes. A pipeline built on separate Seen + Record
// calls leaves the race window open and fails this drill loudly.
//
// # Failing closed
//
// A store failure means the duplicate state is unknown. [Fail] makes every
// store call return an error; [RequireFailClosed] then asserts the dispatch
// errors and the side effect never runs. Set Failing.Err to nil mid-test to
// model the store healing.
//
// # The TTL window
//
// [RequireAllowsAfterExpiry] asserts the caller-visible half of a TTL: after
// the window passes, a retry legitimately re-processes. The drill polls with
// a bounded deadline instead of sleeping a fixed margin — pass a small TTL
// (single-digit milliseconds) to keep it fast.
//
// # Driving the HTTP adapter
//
//	exec := idemtest.NewExecutions()
//	handler := middleware.HTTP(store, time.Minute, idemtest.Counting(exec, app))
//
//	first := idemtest.Send(t, handler, "order-42:place")
//	idemtest.RequireStatus(t, first, http.StatusCreated)
//	idemtest.RequireExecutions(t, exec, 1)
//
//	retry := idemtest.Send(t, handler, "order-42:place")
//	idemtest.RequireDuplicate(t, retry)
//	idemtest.RequireExecutions(t, exec, 1)
//
// [SendConcurrently] releases N same-key requests at once (the responses come
// back in completion order — count statuses, do not index them). An empty key
// sends no header, driving the 400 path; [WithMethod], [WithBody], and
// [WithHeader] shape the request.
//
// # Watching the wiring
//
// [NewRecorder] spies on every store call, so key-derivation and TTL bugs
// surface as assertion failures ([RequireClaimed]) instead of silences.
//
// # Implementing a backend instead?
//
// The drills exercise your integration; the reusable suite that verifies a
// Store implementation against the full invariant set lives in
// github.com/larsartmann/go-idempotency/contract. The two compose: run
// contract.RunTests for the backend, the idemtest drills for everything above
// it.
package idemtest
