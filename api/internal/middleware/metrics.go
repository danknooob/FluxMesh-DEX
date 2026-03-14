package middleware

import (
	"net/http"
	"time"

	"github.com/danknooob/fluxmesh-dex/api/internal/metrics"
)

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Metrics records request count, duration, and status for Prometheus.
// Skips /metrics and /health to avoid cardinality.
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "" {
			path = "/"
		}
		if path == "/metrics" || path == "/health" {
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		metrics.ObserveHTTP(r.Method, path, rw.status, time.Since(start))
	})
}
