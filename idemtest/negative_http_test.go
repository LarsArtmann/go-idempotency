package idemtest_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/larsartmann/go-idempotency/idemtest"
)

// The HTTP-drill sabotage proofs: each assertion must be shown to fail
// against the violation it names, and the failure message must name it.

func recorderWithStatus(status int) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	rec.WriteHeader(status)

	return rec
}

func TestNegative_RequireStatusDetectsMismatch(t *testing.T) {
	t.Parallel()

	expectDrillFailure(t, "a status mismatch", "status = 418", func(tb testing.TB) {
		tb.Helper()

		idemtest.RequireStatus(tb, recorderWithStatus(http.StatusTeapot), http.StatusCreated)
	})
}

func TestNegative_RequireDuplicateDetectsProcessedRetry(t *testing.T) {
	t.Parallel()

	expectDrillFailure(t, "a duplicate retry that was processed", "duplicate retry was not rejected", func(tb testing.TB) {
		tb.Helper()

		idemtest.RequireDuplicate(tb, recorderWithStatus(http.StatusCreated))
	})
}

func TestNegative_RequireMissingKeyRejectedDetectsSilentProcessing(t *testing.T) {
	t.Parallel()

	expectDrillFailure(t, "a missing key that was processed", "missing key was not rejected", func(tb testing.TB) {
		tb.Helper()

		idemtest.RequireMissingKeyRejected(tb, recorderWithStatus(http.StatusOK))
	})
}

func TestNegative_RequireStoreUnavailableDetectsWrongFailureStatus(t *testing.T) {
	t.Parallel()

	expectDrillFailure(t, "a store failure surfaced as the wrong status", "store failure was not surfaced", func(tb testing.TB) {
		tb.Helper()

		idemtest.RequireStoreUnavailable(tb, recorderWithStatus(http.StatusInternalServerError))
	})
}
