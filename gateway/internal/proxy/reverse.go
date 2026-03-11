package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

// New returns a reverse proxy that forwards requests to target.
// It preserves the original path (no rewriting).
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
	return rp
}
