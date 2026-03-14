package main

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danknooob/fluxmesh-dex/api/internal/auth"
	"github.com/danknooob/fluxmesh-dex/api/internal/config"
	"github.com/danknooob/fluxmesh-dex/api/internal/dbseed"
	"github.com/danknooob/fluxmesh-dex/api/internal/handler"
	"github.com/danknooob/fluxmesh-dex/api/internal/kafka"
	"github.com/danknooob/fluxmesh-dex/api/internal/logger"
	"github.com/danknooob/fluxmesh-dex/api/internal/metrics"
	"github.com/danknooob/fluxmesh-dex/api/internal/middleware"
	"github.com/danknooob/fluxmesh-dex/api/internal/migrations"
	"github.com/danknooob/fluxmesh-dex/api/internal/models"
	"github.com/danknooob/fluxmesh-dex/api/internal/repository"
	"github.com/danknooob/fluxmesh-dex/api/internal/service"
	"github.com/danknooob/fluxmesh-dex/api/internal/tracing"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/riandyrn/otelchi"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	svcLogger := logger.New("api")
	slog.SetDefault(svcLogger)

	ctx := context.Background()
	shutdown, err := tracing.Init(ctx, "api")
	if err != nil {
		slog.Error("tracing init failed", "error", err)
	} else {
		defer shutdown()
	}

	cfg := config.Load()

	db, err := gorm.Open(postgres.Open(cfg.DB.DSN), &gorm.Config{})
	if err != nil {
		slog.Error("db connect failed", "error", err)
		return
	}
	if err := db.AutoMigrate(&models.User{}, &models.Order{}, &models.Market{}, &models.Balance{}); err != nil {
		slog.Error("migrate failed", "error", err)
		return
	}
	if err := migrations.RunStoredProcedures(db); err != nil {
		slog.Error("stored procedures failed", "error", err)
		return
	}

	if err := dbseed.SeedInitialMarkets(db); err != nil {
		slog.Warn("seed markets", "error", err)
	}
	dbseed.SeedDefaultUsers(db)

	producer := kafka.NewProducer(cfg.Kafka.Brokers)
	defer producer.Close()

	userRepo := repository.NewUserRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	marketRepo := repository.NewMarketRepository(db)
	balanceRepo := repository.NewBalanceRepository(db)

	userSvc := service.NewUserService(userRepo, producer)
	marketSvc := service.NewMarketService(marketRepo)
	orderSvc := service.NewOrderService(orderRepo, marketSvc, producer)

	authCtrl := handler.NewAuthController(cfg, userSvc)
	profileCtrl := handler.NewProfileController(userSvc)
	orderCtrl := handler.NewOrderController(orderSvc)
	marketCtrl := handler.NewMarketController(marketSvc)
	balanceCtrl := handler.NewBalanceController(balanceRepo)

	r := chi.NewRouter()
	r.Use(otelchi.Middleware("api"))
	r.Use(chimw.StripSlashes)
	r.Use(chimw.RequestID)
	r.Use(middleware.Metrics)

	r.Handle("/metrics", metrics.Handler())
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	r.Post("/auth/login", authCtrl.Login)
	r.Post("/auth/register", authCtrl.Register)

	r.Group(func(gr chi.Router) {
		gr.Use(auth.GatewayMiddleware)

		gr.Get("/profile", profileCtrl.Get)
		gr.Put("/profile", profileCtrl.Update)
		gr.Delete("/profile", profileCtrl.Delete)

		gr.Get("/orders", orderCtrl.List)
		gr.Post("/orders", orderCtrl.Create)
		gr.Delete("/orders/{id}", orderCtrl.Delete)

		gr.Get("/markets", marketCtrl.List)
		gr.Get("/markets/{id}", marketCtrl.Get)
		gr.Get("/markets/{id}/depth", orderCtrl.Depth)

		gr.Get("/balances", balanceCtrl.List)
	})

	slog.Info("api started", "port", cfg.HTTPPort, "metrics", "http://localhost:"+cfg.HTTPPort+"/metrics")
	if err := http.ListenAndServe(":"+cfg.HTTPPort, r); err != nil {
		slog.Error("serve failed", "error", err)
	}
}
