package idemtest_test

import (
	"errors"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-idempotency"
	"github.com/larsartmann/go-idempotency/idemtest"
	"github.com/larsartmann/go-idempotency/middleware"
)

func newAppHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
}

func newHTTPHandler(store idempotency.Store, exec *idemtest.Executions) http.Handler {
	return middleware.HTTP(store, time.Minute, idemtest.Counting(exec, newAppHandler()))
}

func TestSend_FirstRequestProcesses(t *testing.T) {
	t.Parallel()

	exec := idemtest.NewExecutions()
	handler := newHTTPHandler(idemtest.NewStore(), exec)

	rec := idemtest.Send(t, handler, "order-42:place")

	idemtest.RequireStatus(t, rec, http.StatusCreated)
	idemtest.RequireExecutions(t, exec, 1)
}

func TestSend_DuplicateIsRejected(t *testing.T) {
	t.Parallel()

	exec := idemtest.NewExecutions()
	handler := newHTTPHandler(idemtest.NewStore(), exec)

	idemtest.RequireStatus(t, idemtest.Send(t, handler, "order-42:place"), http.StatusCreated)
	idemtest.RequireDuplicate(t, idemtest.Send(t, handler, "order-42:place"))
	idemtest.RequireExecutions(t, exec, 1)
}

func TestSend_EmptyKeyOmitsHeader(t *testing.T) {
	t.Parallel()

	exec := idemtest.NewExecutions()
	handler := newHTTPHandler(idemtest.NewStore(), exec)

	idemtest.RequireMissingKeyRejected(t, idemtest.Send(t, handler, ""))
	idemtest.RequireExecutions(t, exec, 0)
}

func TestSend_StoreFailureIsUnavailable(t *testing.T) {
	t.Parallel()

	exec := idemtest.NewExecutions()
	store := idemtest.Fail(idemtest.NewStore(), errors.New("redis down")) //nolint:err113 // test sentinel
	handler := newHTTPHandler(store, exec)

	idemtest.RequireStoreUnavailable(t, idemtest.Send(t, handler, "order-42:place"))
	idemtest.RequireExecutions(t, exec, 0)
}

func TestSend_KeysAreIndependent(t *testing.T) {
	t.Parallel()

	exec := idemtest.NewExecutions()
	handler := newHTTPHandler(idemtest.NewStore(), exec)

	idemtest.RequireStatus(t, idemtest.Send(t, handler, "key-a"), http.StatusCreated)
	idemtest.RequireStatus(t, idemtest.Send(t, handler, "key-b"), http.StatusCreated)
	idemtest.RequireExecutions(t, exec, 2)
}

func TestSend_RequestOptions(t *testing.T) {
	t.Parallel()

	var seenMethod, seenBody, seenHeader string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenMethod = r.Method

		body, readErr := io.ReadAll(r.Body)
		if readErr != nil {
			t.Errorf("read body: %v", readErr)
		}

		seenBody = string(body)
		seenHeader = r.Header.Get("X-Trace")

		w.WriteHeader(http.StatusOK)
	})

	rec := idemtest.Send(t, handler, "key",
		idemtest.WithMethod(http.MethodPut),
		idemtest.WithBody(`{"order":42}`),
		idemtest.WithHeader("X-Trace", "abc"),
	)

	idemtest.RequireStatus(t, rec, http.StatusOK)

	if seenMethod != http.MethodPut || seenBody != `{"order":42}` || seenHeader != "abc" {
		t.Fatalf("request = (%s, %q, %q), want (PUT, order body, abc)", seenMethod, seenBody, seenHeader)
	}
}

func TestSendConcurrently_ExactlyOneProcesses(t *testing.T) {
	t.Parallel()

	exec := idemtest.NewExecutions()
	handler := newHTTPHandler(idemtest.NewStore(), exec)

	const requests = 50

	recs := idemtest.SendConcurrently(t, handler, "contended-key", requests)

	conflicts := 0

	for _, rec := range recs {
		switch rec.Code {
		case http.StatusCreated:
		case http.StatusConflict:
			conflicts++
		default:
			t.Fatalf("status = %d, want 201 or 409", rec.Code)
		}
	}

	if conflicts != requests-1 {
		t.Fatalf("conflicts = %d, want %d", conflicts, requests-1)
	}

	idemtest.RequireExecutions(t, exec, 1)
}

func TestSendConcurrently_RequestOptions(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex

	bodies := make([]string, 0, 8)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, readErr := io.ReadAll(r.Body)
		if readErr != nil {
			t.Errorf("read body: %v", readErr)
		}

		mu.Lock()
		bodies = append(bodies, string(body))
		mu.Unlock()

		w.WriteHeader(http.StatusCreated)
	})

	const requests = 8

	recs := idemtest.SendConcurrently(t, handler, "contended-key", requests, idemtest.WithBody(`{"order":42}`))

	for _, rec := range recs {
		idemtest.RequireStatus(t, rec, http.StatusCreated)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(bodies) != requests {
		t.Fatalf("bodies = %d, want %d", len(bodies), requests)
	}

	for _, body := range bodies {
		if body != `{"order":42}` {
			t.Fatalf("request body = %q, want the WithBody payload on every concurrent request", body)
		}
	}
}
