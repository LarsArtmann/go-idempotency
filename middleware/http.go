package middleware

import (
	"errors"
	"net/http"
	"time"

	"github.com/larsartmann/go-idempotency"
)

// HeaderKey is the conventional request header carrying the idempotency key
// (as used by Stripe and most payment APIs).
const HeaderKey = "Idempotency-Key"

// HTTP wraps an http.Handler so each request is processed at most once per
// Idempotency-Key header value for the configured TTL.
//
//   - Missing or empty header → 400 Bad Request (the client contract requires
//     a stable key; silently processing without one defeats the purpose).
//   - Duplicate key → 409 Conflict, no execution. For replay semantics (the
//     retried request should receive the original response instead of a
//     conflict), compose with the response-replay recipe in the package docs
//     of github.com/larsartmann/go-idempotency.
//   - Store failure → 503 Service Unavailable, no execution: fail closed
//     rather than process on unknown state.
//
// Keys are taken verbatim from the header. When several routes share one
// store, namespace the keys (e.g. derive "<route>:<header value>" by
// wrapping the handler) so a key from one endpoint cannot suppress another's.
func HTTP(store idempotency.Store, ttl time.Duration, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get(HeaderKey)
		if key == "" {
			http.Error(w, "missing "+HeaderKey+" header", http.StatusBadRequest)

			return
		}

		err := store.CheckAndRecord(r.Context(), key, ttl)
		switch {
		case err == nil:
			next.ServeHTTP(w, r)
		case errors.Is(err, idempotency.ErrDuplicate):
			http.Error(w, "duplicate request: "+HeaderKey+" already processed", http.StatusConflict)
		case errors.Is(err, idempotency.ErrInvalidTTL):
			// The server chose the TTL, so this is a server bug, not a
			// client error. Fail closed.
			http.Error(w, "idempotency store misconfigured", http.StatusInternalServerError)
		default:
			http.Error(w, "idempotency store unavailable", http.StatusServiceUnavailable)
		}
	})
}
