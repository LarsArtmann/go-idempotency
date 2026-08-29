package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-idempotency"
	"github.com/larsartmann/go-idempotency/middleware"
)

func newTestHandler(store idempotency.Store, handler *atomic.Int64) http.Handler {
	return middleware.HTTP(store, time.Minute, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		handler.Add(1)
		w.WriteHeader(http.StatusCreated)
	}))
}

func TestHTTP_FirstRequestProcesses(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)

	var processed atomic.Int64

	srv := httptest.NewServer(newTestHandler(store, &processed))
	t.Cleanup(srv.Close)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL, http.NoBody)
	req.Header.Set(middleware.HeaderKey, "order-42:place")

	res, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", res.StatusCode)
	}

	if processed.Load() != 1 {
		t.Fatalf("processed = %d, want 1", processed.Load())
	}
}

func TestHTTP_DuplicateRequestConflicts(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)

	var processed atomic.Int64

	srv := httptest.NewServer(newTestHandler(store, &processed))
	t.Cleanup(srv.Close)

	send := func() int {
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL, http.NoBody)
		req.Header.Set(middleware.HeaderKey, "order-42:place")

		res, err := srv.Client().Do(req)
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		defer res.Body.Close()

		return res.StatusCode
	}

	if got := send(); got != http.StatusCreated {
		t.Fatalf("first request status = %d, want 201", got)
	}

	if got := send(); got != http.StatusConflict {
		t.Fatalf("duplicate request status = %d, want 409", got)
	}

	if processed.Load() != 1 {
		t.Fatalf("processed = %d, want 1 (duplicate must not process)", processed.Load())
	}
}

func TestHTTP_MissingHeaderIsBadRequest(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)

	var processed atomic.Int64

	srv := httptest.NewServer(newTestHandler(store, &processed))
	t.Cleanup(srv.Close)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL, http.NoBody)

	res, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", res.StatusCode)
	}

	if processed.Load() != 0 {
		t.Fatalf("processed = %d, want 0", processed.Load())
	}
}

func TestHTTP_StoreErrorIsServiceUnavailable(t *testing.T) {
	t.Parallel()

	store := &failingStore{Store: newTestStore(t), err: errors.New("down")} //nolint:err113 // test sentinel

	var processed atomic.Int64

	srv := httptest.NewServer(newTestHandler(store, &processed))
	t.Cleanup(srv.Close)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL, http.NoBody)
	req.Header.Set(middleware.HeaderKey, "key")

	res, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", res.StatusCode)
	}

	if processed.Load() != 0 {
		t.Fatalf("processed = %d, want 0 (store failure must fail closed)", processed.Load())
	}
}

func TestHTTP_AtMostOnceUnderConcurrency(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)

	var processed atomic.Int64

	srv := httptest.NewServer(newTestHandler(store, &processed))
	t.Cleanup(srv.Close)

	const goroutines = 20

	var (
		wg sync.WaitGroup

		statuses = make(chan int, goroutines)
		started  = make(chan struct{})
	)

	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			<-started

			req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL, http.NoBody)
			req.Header.Set(middleware.HeaderKey, "same-key")

			res, err := srv.Client().Do(req)
			if err != nil {
				t.Errorf("request: %v", err)

				return
			}
			defer res.Body.Close()

			statuses <- res.StatusCode
		}()
	}

	close(started)
	wg.Wait()
	close(statuses)

	created := 0

	for status := range statuses {
		if status == http.StatusCreated {
			created++
		}
	}

	if created != 1 {
		t.Fatalf("created = %d, want exactly 1 across %d concurrent requests", created, goroutines)
	}

	if processed.Load() != 1 {
		t.Fatalf("processed = %d, want 1", processed.Load())
	}
}
