package proxy

import (
	"bytes"
	"io"
	"log/slog"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/danknooob/fluxmesh-dex/gateway/internal/metrics"
	"github.com/sony/gobreaker"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

const (
	maxRetries = 2
	baseDelay  = 150 * time.Millisecond
	maxDelay   = 2 * time.Second
)

// Default circuit breaker: open after 5 consecutive failures, try again after 30s.
const (
	cbMaxRequests   = 1
	cbTimeout       = 30 * time.Second
	cbReadyToTrip   = 5 // consecutive failures before opening
)

// New returns a reverse proxy that forwards requests to target with automatic
// retry on transient upstream failures (connection refused, 502, 503, 504)
// and a circuit breaker so sustained upstream failures stop hammering the backend.
// Only idempotent methods (GET/HEAD/OPTIONS) are retried on HTTP-level errors;
// network-level failures (connection refused) are retried for all methods
// because the request never reached the upstream.
func New(target string) http.Handler {
	u, err := url.Parse(target)
	if err != nil {
		panic("invalid proxy target: " + err.Error())
	}
	rt := &retryTransport{base: http.DefaultTransport, target: u}
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "gateway-proxy-" + u.Host,
		MaxRequests: cbMaxRequests,
		Timeout:     cbTimeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= cbReadyToTrip
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			slog.Info("circuit breaker state change", "name", name, "from", from.String(), "to", to.String())
			metrics.SetCircuitBreakerState(u.Host, to == gobreaker.StateOpen)
		},
	})
	rp := httputil.NewSingleHostReverseProxy(u)
	origDirector := rp.Director
	backend := u.Host
	rp.Director = func(req *http.Request) {
		origDirector(req)
		req.Host = u.Host
		// Propagate trace context to backend for distributed tracing
		otel.GetTextMapPropagator().Inject(req.Context(), propagation.HeaderCarrier(req.Header))
	}
	rp.ModifyResponse = retryableResponseCheck
	rp.Transport = &circuitBreakerTransport{base: rt, cb: cb, backend: backend}
	return rp
}

// circuitBreakerTransport wraps a RoundTripper with a circuit breaker.
// When the circuit is open, RoundTrip returns immediately with gobreaker.ErrOpen
// so the gateway can fail fast instead of hammering a failing upstream.
type circuitBreakerTransport struct {
	base    http.RoundTripper
	cb      *gobreaker.CircuitBreaker
	backend string
}

func (t *circuitBreakerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	res, err := t.cb.Execute(func() (interface{}, error) {
		return t.base.RoundTrip(req)
	})
	duration := time.Since(start)
	method := req.Method
	path := req.URL.Path
	if path == "" {
		path = "/"
	}
	if err != nil {
		metrics.ObserveHTTP(t.backend, method, path, 0, duration)
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	metrics.ObserveHTTP(t.backend, method, path, res.(*http.Response).StatusCode, duration)
	return res.(*http.Response), nil
}

// retryableResponseCheck marks 502/503/504 responses so the retry transport
// can see them (the transport itself handles the loop).
func retryableResponseCheck(resp *http.Response) error {
	return nil
}

type retryTransport struct {
	base   http.RoundTripper
	target *url.URL
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	idempotent := isIdempotent(req.Method)
	var lastResp *http.Response
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		resp, err := t.base.RoundTrip(req)

		if err != nil {
			lastErr = err
			if isNetworkError(err) && attempt < maxRetries {
				delay := backoff(attempt)
				slog.Warn("gateway-proxy network error, retrying", "attempt", attempt+1, "max_retries", maxRetries, "delay", delay, "error", err)
				time.Sleep(delay)
				continue
			}
			return nil, err
		}

		if isRetryableStatus(resp.StatusCode) && idempotent && attempt < maxRetries {
			delay := backoff(attempt)
			slog.Warn("gateway-proxy upstream error, retrying", "status", resp.StatusCode, "method", req.Method, "path", req.URL.Path, "attempt", attempt+1, "max_retries", maxRetries, "delay", delay)
			_ = resp.Body.Close()
			time.Sleep(delay)
			continue
		}

		return resp, nil
	}

	if lastResp != nil {
		return lastResp, nil
	}
	return nil, lastErr
}

func isIdempotent(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	return false
}

func isRetryableStatus(code int) bool {
	return code == 502 || code == 503 || code == 504
}

func isNetworkError(err error) bool {
	if _, ok := err.(*net.OpError); ok {
		return true
	}
	if _, ok := err.(net.Error); ok {
		return true
	}
	return false
}

func backoff(attempt int) time.Duration {
	d := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt)))
	if d > maxDelay {
		d = maxDelay
	}
	jitter := time.Duration(rand.Int63n(int64(baseDelay)))
	return d + jitter
}
