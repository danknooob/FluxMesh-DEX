package proxy

import (
	"bytes"
	"io"
	"log"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

const (
	maxRetries = 2
	baseDelay  = 150 * time.Millisecond
	maxDelay   = 2 * time.Second
)

// New returns a reverse proxy that forwards requests to target with automatic
// retry on transient upstream failures (connection refused, 502, 503, 504).
// Only idempotent methods (GET/HEAD/OPTIONS) are retried on HTTP-level errors;
// network-level failures (connection refused) are retried for all methods
// because the request never reached the upstream.
func New(target string) http.Handler {
	u, err := url.Parse(target)
	if err != nil {
		panic("invalid proxy target: " + err.Error())
	}
	rp := httputil.NewSingleHostReverseProxy(u)
	origDirector := rp.Director
	rp.Director = func(req *http.Request) {
		origDirector(req)
		req.Host = u.Host
	}
	rp.ModifyResponse = retryableResponseCheck
	rp.Transport = &retryTransport{base: http.DefaultTransport, target: u}
	return rp
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
				log.Printf("gateway-proxy: network error (attempt %d/%d), retrying in %v: %v",
					attempt+1, maxRetries, delay, err)
				time.Sleep(delay)
				continue
			}
			return nil, err
		}

		if isRetryableStatus(resp.StatusCode) && idempotent && attempt < maxRetries {
			delay := backoff(attempt)
			log.Printf("gateway-proxy: %d on %s %s (attempt %d/%d), retrying in %v",
				resp.StatusCode, req.Method, req.URL.Path, attempt+1, maxRetries, delay)
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
