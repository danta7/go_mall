package middleware

import (
	"github.com/google/uuid"
	"net/http"
	"strings"
)

const (
	HeaderRequestID = "X-Request-ID" // 请求头中的 key
)

// RequestID ensures each request han an ID in context and response header
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get(HeaderRequestID)
		if strings.TrimSpace(rid) == "" {
			rid = uuid.New().String()
		}
		w.Header().Set(HeaderRequestID, rid)
		next.ServeHTTP(w, r.WithContext(withRequestID(r.Context(), rid)))
	})
}
