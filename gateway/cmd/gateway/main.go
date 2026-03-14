package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/danknooob/fluxmesh-dex/gateway/internal/logger"
	"github.com/danknooob/fluxmesh-dex/gateway/internal/metrics"
	"github.com/danknooob/fluxmesh-dex/gateway/internal/middleware"
	"github.com/danknooob/fluxmesh-dex/gateway/internal/proxy"
	"github.com/danknooob/fluxmesh-dex/gateway/internal/swagger"
	"github.com/danknooob/fluxmesh-dex/gateway/internal/tracing"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/riandyrn/otelchi"
)

func main() {
	svcLogger := logger.New("gateway")
	slog.SetDefault(svcLogger)

	ctx := context.Background()
	shutdown, err := tracing.Init(ctx, "gateway")
	if err != nil {
		slog.Error("tracing init failed", "error", err)
	} else {
		defer shutdown()
	}

	port := getEnv("GATEWAY_PORT", "8000")
	jwtSecret := getEnv("JWT_SECRET", "change-me-in-production")
	apiTarget := getEnv("API_TARGET", "http://localhost:8080")
	controlTarget := getEnv("CONTROL_TARGET", "http://localhost:8081")

	apiProxy := proxy.New(apiTarget)
	controlProxy := proxy.New(controlTarget)

	rl := middleware.NewRateLimiter(20, 40) // 20 req/s per client, burst 40

	r := chi.NewRouter()
	r.Use(otelchi.Middleware("gateway"))
	r.Use(chimw.StripSlashes)
	r.Use(chimw.RealIP)
	r.Use(chimw.RequestID)
	r.Use(chimw.Recoverer)

	// Metrics — no auth (scrape by Prometheus)
	r.Handle("/metrics", metrics.Handler())

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
		gr.Get("/markets/{id}/depth", apiProxy.ServeHTTP)
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

	slog.Info("gateway started", "port", port, "api_target", apiTarget, "control_target", controlTarget, "docs", "http://localhost:"+port+"/docs", "metrics", "http://localhost:"+port+"/metrics")

	srv := &http.Server{Addr: ":" + port, Handler: r}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("gateway server error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("gateway shutting down")
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("gateway shutdown error", "error", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
