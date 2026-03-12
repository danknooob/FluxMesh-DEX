package main

import (
	"log"
	"net/http"
	"os"

	"github.com/danknooob/fluxmesh-dex/gateway/internal/middleware"
	"github.com/danknooob/fluxmesh-dex/gateway/internal/proxy"
	"github.com/danknooob/fluxmesh-dex/gateway/internal/swagger"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

func main() {
	port := getEnv("GATEWAY_PORT", "8000")
	jwtSecret := getEnv("JWT_SECRET", "change-me-in-production")
	apiTarget := getEnv("API_TARGET", "http://localhost:8080")
	controlTarget := getEnv("CONTROL_TARGET", "http://localhost:8081")

	apiProxy := proxy.New(apiTarget)
	controlProxy := proxy.New(controlTarget)

	rl := middleware.NewRateLimiter(20, 40) // 20 req/s per client, burst 40

	r := chi.NewRouter()
	r.Use(chimw.StripSlashes)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	// Swagger UI — public, no auth
	specPath := getEnv("SWAGGER_SPEC", "../docs/swagger.yaml")
	swag := swagger.New(specPath)
	r.Get("/docs", swag.UI)
	r.Get("/docs/swagger.yaml", swag.Spec)

	// Public — auth routes do not need JWT or rate limiting
	r.Post("/auth/login", apiProxy.ServeHTTP)
	r.Post("/auth/register", apiProxy.ServeHTTP)

	// Authenticated trader routes — JWT + rate limit, then proxy to API
	r.Group(func(gr chi.Router) {
		gr.Use(middleware.JWTAuth(jwtSecret, false))
		gr.Use(rl.Handler)

		gr.Get("/profile", apiProxy.ServeHTTP)
		gr.Put("/profile", apiProxy.ServeHTTP)
		gr.Delete("/profile", apiProxy.ServeHTTP)

		gr.Get("/markets", apiProxy.ServeHTTP)
		gr.Get("/markets/{id}", apiProxy.ServeHTTP)
		gr.Get("/orders", apiProxy.ServeHTTP)
		gr.Post("/orders", apiProxy.ServeHTTP)
		gr.Delete("/orders/{id}", apiProxy.ServeHTTP)
		gr.Get("/balances", apiProxy.ServeHTTP)
	})

	// Admin routes — JWT (admin-only) + rate limit, then proxy to control plane
	r.Group(func(gr chi.Router) {
		gr.Use(middleware.JWTAuth(jwtSecret, true))
		gr.Use(rl.Handler)

		gr.Get("/admin/*", controlProxy.ServeHTTP)
		gr.Post("/admin/*", controlProxy.ServeHTTP)
		gr.Put("/admin/*", controlProxy.ServeHTTP)
		gr.Delete("/admin/*", controlProxy.ServeHTTP)
	})

	log.Printf("API Gateway listening on :%s", port)
	log.Printf("  -> API backend:     %s", apiTarget)
	log.Printf("  -> Control backend: %s", controlTarget)
	log.Printf("  -> Swagger UI:      http://localhost:%s/docs", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("gateway: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
