package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// HTTPRequestTotal counts requests by backend, method, path, status.
	HTTPRequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_http_requests_total",
			Help: "Total HTTP requests proxied by backend, method, path, status",
		},
		[]string{"backend", "method", "path", "status"},
	)
	// HTTPRequestDurationSeconds is the request duration histogram.
	HTTPRequestDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gateway_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 14),
		},
		[]string{"backend", "method", "path"},
	)
	// CircuitBreakerState is 1 when circuit is open, 0 when closed.
	CircuitBreakerState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gateway_circuit_breaker_open",
			Help: "1 if circuit breaker is open for this backend, 0 otherwise",
		},
		[]string{"backend"},
	)
)

// ObserveHTTP records a proxied request for Prometheus.
func ObserveHTTP(backend, method, path string, status int, duration time.Duration) {
	statusStr := strconv.Itoa(status)
	HTTPRequestTotal.WithLabelValues(backend, method, path, statusStr).Inc()
	HTTPRequestDurationSeconds.WithLabelValues(backend, method, path).Observe(duration.Seconds())
}

// SetCircuitBreakerState updates the circuit breaker gauge (1=open, 0=closed/half-open for simplicity).
func SetCircuitBreakerState(backend string, open bool) {
	v := 0.0
	if open {
		v = 1.0
	}
	CircuitBreakerState.WithLabelValues(backend).Set(v)
}

// Handler returns an http.Handler for the /metrics endpoint.
func Handler() http.Handler {
	return promhttp.Handler()
}
