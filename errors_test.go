package idempotency_test

// Public-API contract locks for the error sentinels: family membership
// (Conflict / Rejection), retryability, and errors.Is matching through
// wrapping. These pin the semantics downstream HTTP mapping relies on.

import (
	"errors"
	"fmt"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/go-idempotency"
)

func TestErrDuplicate_IsConflict(t *testing.T) {
	t.Parallel()

	if fam := errorfamily.Classify(idempotency.ErrDuplicate); fam != errorfamily.Conflict {
		t.Fatalf("family: want Conflict, got %s", fam)
	}

	if errorfamily.IsRetryable(idempotency.ErrDuplicate) {
		t.Fatal("Conflict must not be retryable")
	}
}

func TestErrInvalidTTL_IsRejection(t *testing.T) {
	t.Parallel()

	if fam := errorfamily.Classify(idempotency.ErrInvalidTTL); fam != errorfamily.Rejection {
		t.Fatalf("family: want Rejection, got %s", fam)
	}

	if errorfamily.IsRetryable(idempotency.ErrInvalidTTL) {
		t.Fatal("Rejection must not be retryable")
	}
}

// TestErrClassification_SurvivesWrapping locks the family membership of both
// sentinels as an API contract: middleware and handlers may wrap store errors
// on their way out, and downstream HTTP mapping must still see Conflict and
// Rejection through the wrapping.
func TestErrClassification_SurvivesWrapping(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		sentinel error
		family   errorfamily.Family
	}{
		{name: "ErrDuplicate", sentinel: idempotency.ErrDuplicate, family: errorfamily.Conflict},
		{name: "ErrInvalidTTL", sentinel: idempotency.ErrInvalidTTL, family: errorfamily.Rejection},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			wrapped := fmt.Errorf("dispatch: %w", fmt.Errorf("handler: %w", testCase.sentinel))

			if fam := errorfamily.Classify(wrapped); fam != testCase.family {
				t.Fatalf("wrapped family: want %s, got %s", testCase.family, fam)
			}

			if errorfamily.IsRetryable(wrapped) {
				t.Fatalf("wrapped %s must not be retryable", testCase.name)
			}
		})
	}
}

func TestSentinels_SurviveErrorWrapping(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		sentinel error
	}{
		{name: "ErrDuplicate", sentinel: idempotency.ErrDuplicate},
		{name: "ErrInvalidTTL", sentinel: idempotency.ErrInvalidTTL},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			wrapped := fmt.Errorf("dispatch: %w", fmt.Errorf("handler: %w", testCase.sentinel))

			if !errors.Is(wrapped, testCase.sentinel) {
				t.Fatalf(
					"errors.Is must match through fmt.Errorf wrapping (code=%q)",
					errorfamily.Code(testCase.sentinel),
				)
			}
		})
	}
}
