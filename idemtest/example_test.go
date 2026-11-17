package idemtest_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/larsartmann/go-idempotency"
	"github.com/larsartmann/go-idempotency/idemtest"
)

// ExampleNewStore shows the blessed test double: import idemtest from a
// _test.go file, get a correct in-memory Store, and swap nothing else in your
// wiring. The drills and the middleware both accept it unchanged.
func ExampleNewStore() {
	store := idemtest.NewStore()
	ctx := context.Background()

	first := store.CheckAndRecord(ctx, "order-42:place", 10*time.Minute)
	duplicate := store.CheckAndRecord(ctx, "order-42:place", 10*time.Minute)

	fmt.Println("first claim:", first == nil)
	fmt.Println("retry:", errors.Is(duplicate, idempotency.ErrDuplicate))

	// Output:
	// first claim: true
	// retry: true
}

// ExampleFail shows failure injection: with every store call failing, the
// wrapped dispatch must fail closed (middleware.NewCommand refuses to execute)
// because the duplicate state is unknown.
func ExampleFail() {
	store := idemtest.Fail(idemtest.NewStore(), errors.New("redis down"))

	err := store.CheckAndRecord(context.Background(), "order-42:place", time.Minute)

	fmt.Println("store down:", err != nil)
	fmt.Println("not a sentinel:",
		!errors.Is(err, idempotency.ErrDuplicate) && !errors.Is(err, idempotency.ErrInvalidTTL))

	// Output:
	// store down: true
	// not a sentinel: true
}

// ExampleNewRecorder shows the spy: key-derivation and TTL bugs surface as
// readable call records instead of silences.
func ExampleNewRecorder() {
	recorder := idemtest.NewRecorder(idemtest.NewStore())
	ctx := context.Background()

	_ = recorder.CheckAndRecord(ctx, "order-42:place", time.Minute)

	for _, call := range recorder.Calls() {
		fmt.Printf("%s %s %s\n", call.Op, call.Key, call.TTL)
	}

	// Output:
	// CheckAndRecord order-42:place 1m0s
}

// ExampleCounting shows the HTTP execution adapter: one counter, wrapped
// around the app handler, observed through RequireExecutions in real tests.
func ExampleCounting() {
	executions := idemtest.NewExecutions()

	counted := idemtest.Counting(executions, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	request, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"http://example.test/orders",
		http.NoBody,
	)
	if err != nil {
		panic(err)
	}

	counted.ServeHTTP(httptest.NewRecorder(), request)

	fmt.Println("processed:", executions.Total())

	// Output:
	// processed: 1
}
