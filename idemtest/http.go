package idemtest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/larsartmann/go-idempotency/middleware"
)

// RequestOption customizes the request a [Send] drill issues.
type RequestOption func(*requestConfig)

type requestConfig struct {
	method  string
	body    string
	headers map[string]string
}

func applyRequestOptions(opts []RequestOption) requestConfig {
	cfg := requestConfig{ //nolint:exhaustruct_v5 // body may stay empty
		method:  http.MethodPost,
		headers: map[string]string{},
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	return cfg
}

// WithMethod sends method instead of the default POST.
func WithMethod(method string) RequestOption {
	return func(cfg *requestConfig) { cfg.method = method }
}

// WithBody sends body as the request body.
func WithBody(body string) RequestOption {
	return func(cfg *requestConfig) { cfg.body = body }
}

// WithHeader adds a request header, replacing any previous value for the key.
func WithHeader(key, value string) RequestOption {
	return func(cfg *requestConfig) { cfg.headers[key] = value }
}

// Send drives one request against handler with the given idempotency key and
// returns the recorded response. An empty key sends no Idempotency-Key header
// at all — the missing-key path.
//
// The request runs in-process through httptest.ResponseRecorder: no server,
// no socket, because idempotency responses are single-shot (ADR-007). The
// default request is a bodiless POST; see [WithMethod], [WithBody], and
// [WithHeader].
func Send(tb testing.TB, handler http.Handler, key string, opts ...RequestOption) *httptest.ResponseRecorder {
	tb.Helper()

	return send(handler, key, applyRequestOptions(opts))
}

// SendConcurrently drives goroutines requests with the same idempotency key,
// released simultaneously — the shape of retries racing the first delivery —
// and returns each recorded response in completion order (unordered: count
// statuses, do not index them).
func SendConcurrently(
	tb testing.TB,
	handler http.Handler,
	key string,
	goroutines int,
) []*httptest.ResponseRecorder {
	tb.Helper()

	if goroutines < 1 {
		tb.Fatalf("goroutines = %d, want at least 1", goroutines)

		return nil
	}

	var (
		mu   sync.Mutex
		recs []*httptest.ResponseRecorder
		wg   sync.WaitGroup
	)

	started := make(chan struct{})
	cfg := applyRequestOptions(nil)

	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			<-started

			rec := send(handler, key, cfg)

			appendRecording(&mu, &recs, rec)
		}()
	}

	close(started)
	wg.Wait()

	return recs
}

func appendRecording(
	mu *sync.Mutex,
	recs *[]*httptest.ResponseRecorder,
	rec *httptest.ResponseRecorder,
) {
	mu.Lock()
	defer mu.Unlock()

	*recs = append(*recs, rec)
}

// RequireStatus fails tb unless the recorded response carries the want status.
func RequireStatus(tb testing.TB, rec *httptest.ResponseRecorder, want int) {
	tb.Helper()

	if rec.Code != want {
		tb.Fatalf("status = %d, want %d (body: %q)", rec.Code, want, strings.TrimSpace(rec.Body.String()))
	}
}

// RequireDuplicate fails tb unless the response is the HTTP adapter's 409
// Conflict: a live claim exists and the request was not processed.
func RequireDuplicate(tb testing.TB, rec *httptest.ResponseRecorder) {
	tb.Helper()

	if rec.Code != http.StatusConflict {
		tb.Fatalf("status = %d, want %d Conflict: the duplicate retry was not rejected", rec.Code, http.StatusConflict)
	}
}

// RequireMissingKeyRejected fails tb unless the response is the HTTP adapter's
// 400 Bad Request for a missing or empty Idempotency-Key header: silently
// processing without a key defeats the client contract.
func RequireMissingKeyRejected(tb testing.TB, rec *httptest.ResponseRecorder) {
	tb.Helper()

	if rec.Code != http.StatusBadRequest {
		tb.Fatalf(
			"status = %d, want %d Bad Request: the missing key was not rejected",
			rec.Code, http.StatusBadRequest,
		)
	}
}

// RequireStoreUnavailable fails tb unless the response is the HTTP adapter's
// 503 Service Unavailable: a store failure means unknown duplicate state, and
// the adapter must fail closed rather than process.
func RequireStoreUnavailable(tb testing.TB, rec *httptest.ResponseRecorder) {
	tb.Helper()

	if rec.Code != http.StatusServiceUnavailable {
		tb.Fatalf(
			"status = %d, want %d Service Unavailable: the store failure was not surfaced",
			rec.Code, http.StatusServiceUnavailable,
		)
	}
}

func send(handler http.Handler, key string, cfg requestConfig) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(
		context.Background(),
		cfg.method,
		"/",
		strings.NewReader(cfg.body),
	)

	if key != "" {
		req.Header.Set(middleware.HeaderKey, key)
	}

	for headerKey, headerValue := range cfg.headers {
		req.Header.Set(headerKey, headerValue)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	return rec
}
