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
	// HTTPRequestTotal counts API requests by method, path, status.
	HTTPRequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_http_requests_total",
			Help: "Total HTTP requests by method, path, status",
		},
		[]string{"method", "path", "status"},
	)
	// HTTPRequestDurationSeconds is the request duration histogram.
	HTTPRequestDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "api_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 14),
		},
		[]string{"method", "path"},
	)
	// KafkaProducerMessagesTotal counts published messages by topic and status (ok, error).
	KafkaProducerMessagesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_kafka_producer_messages_total",
			Help: "Total Kafka messages published by topic and status",
		},
		[]string{"topic", "status"},
	)
)

// ObserveHTTP records an API request for Prometheus.
func ObserveHTTP(method, path string, status int, duration time.Duration) {
	if path == "" {
		path = "/"
	}
	statusStr := strconv.Itoa(status)
	HTTPRequestTotal.WithLabelValues(method, path, statusStr).Inc()
	HTTPRequestDurationSeconds.WithLabelValues(method, path).Observe(duration.Seconds())
}

// ObserveKafkaPublish records a Kafka publish for Prometheus.
func ObserveKafkaPublish(topic string, err error) {
	status := "ok"
	if err != nil {
		status = "error"
	}
	KafkaProducerMessagesTotal.WithLabelValues(topic, status).Inc()
}

// Handler returns the Prometheus /metrics handler.
func Handler() http.Handler {
	return promhttp.Handler()
}
