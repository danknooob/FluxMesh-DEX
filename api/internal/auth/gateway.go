package auth

import "net/http"

// GatewayMiddleware trusts X-User-ID and X-Role headers injected by the API
// gateway after it has already validated the JWT. If the headers are missing
// (direct access bypassing gateway), the request is rejected.
func GatewayMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-User-ID")
		role := r.Header.Get("X-Role")

		if userID == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := WithUser(r.Context(), userID, role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
