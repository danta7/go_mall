package middleware

import (
	"go.uber.org/zap"
	"net/http"
	"time"
)

// AccessLog logs basic HTTP access information.
func AccessLog(logger *zap.Logger) func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWrite{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rw, r)
			dur := time.Since(start)
			logger.Info("http_access",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", rw.statusCode),
				zap.Duration("duration", dur),
				zap.String("request_id", RequestIDFromContext(r.Context())),
			)
		})
	}
}

// 截获状态码
type responseWrite struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWrite) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
