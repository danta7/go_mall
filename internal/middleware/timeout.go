package middleware

import (
	"context"
	"errors"
	"github.com/danta7/go_mall/internal/resp"
	"net/http"
	"time"
)

// Timeout cancels request context after given duration and writes a timeout response
func Timeout(d time.Duration) func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// http.TimeoutHandler writes 503 by default; we intercept context error on write
			next.ServeHTTP(w, r)
		}), d, "")
	}
}

// HandleTimeout is a helper to write unified timeout response when context expired
func HandleTimeout(w http.ResponseWriter, r *http.Request) bool {
	if err := r.Context().Err(); errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		reqID := RequestIDFromContext(r.Context())
		resp.Error(w, resp.HTTPStatusFromCode(resp.CodeTimeout), resp.CodeTimeout, "request timeout", reqID, "")
		return true
	}
	return false
}
