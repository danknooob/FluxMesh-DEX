package main

import (
	"log"
	"net/http"

	"github.com/danknooob/fluxmesh-dex/api/internal/auth"
	"github.com/danknooob/fluxmesh-dex/api/internal/config"
	"github.com/danknooob/fluxmesh-dex/api/internal/dbseed"
	"github.com/danknooob/fluxmesh-dex/api/internal/handler"
	"github.com/danknooob/fluxmesh-dex/api/internal/kafka"
	"github.com/danknooob/fluxmesh-dex/api/internal/models"
	"github.com/danknooob/fluxmesh-dex/api/internal/repository"
	"github.com/danknooob/fluxmesh-dex/api/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()

	db, err := gorm.Open(postgres.Open(cfg.DB.DSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Order{}, &models.Market{}, &models.Balance{}); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	if err := dbseed.SeedInitialMarkets(db); err != nil {
		log.Printf("seed markets: %v", err)
	}
	dbseed.SeedDefaultUsers(db)

	producer := kafka.NewProducer(cfg.Kafka.Brokers)
	defer producer.Close()

	userRepo := repository.NewUserRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	marketRepo := repository.NewMarketRepository(db)

	userSvc := service.NewUserService(userRepo, producer)
	marketSvc := service.NewMarketService(marketRepo)
	orderSvc := service.NewOrderService(orderRepo, marketSvc, producer)

	authCtrl := handler.NewAuthController(cfg, userSvc)
	profileCtrl := handler.NewProfileController(userSvc)
	orderCtrl := handler.NewOrderController(orderSvc)
	marketCtrl := handler.NewMarketController(marketSvc)

	r := chi.NewRouter()
	r.Use(middleware.StripSlashes)

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

		gr.Get("/balances", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("[]"))
		})
	})

	log.Printf("API listening on :%s", cfg.HTTPPort)
	if err := http.ListenAndServe(":"+cfg.HTTPPort, r); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
