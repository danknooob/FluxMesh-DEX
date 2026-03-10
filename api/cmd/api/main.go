package main

import (
	"log"
	"net/http"

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
	if err := db.AutoMigrate(&models.Order{}, &models.Market{}, &models.Balance{}); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	if err := dbseed.SeedInitialMarkets(db); err != nil {
		log.Printf("seed markets: %v", err)
	}

	producer := kafka.NewProducer(cfg.Kafka.Brokers)
	defer producer.Close()

	orderRepo := repository.NewOrderRepository(db)
	marketRepo := repository.NewMarketRepository(db)
	marketSvc := service.NewMarketService(marketRepo)
	orderSvc := service.NewOrderService(orderRepo, marketSvc, producer)

	orderCtrl := handler.NewOrderController(orderSvc)
	marketCtrl := handler.NewMarketController(marketSvc)

	r := chi.NewRouter()
	r.Use(middleware.StripSlashes)

	// Trader-facing APIs
	r.Method(http.MethodGet, "/orders", http.HandlerFunc(orderCtrl.List))
	r.Method(http.MethodPost, "/orders", http.HandlerFunc(orderCtrl.Create))
	r.Method(http.MethodDelete, "/orders/{id}", http.HandlerFunc(orderCtrl.Delete))
	// Support both /markets and /markets/ explicitly to avoid confusion.
	r.Method(http.MethodGet, "/markets", http.HandlerFunc(marketCtrl.List))
	r.Method(http.MethodGet, "/markets/", http.HandlerFunc(marketCtrl.List))
	r.Method(http.MethodGet, "/markets/{id}", http.HandlerFunc(marketCtrl.Get))

	// GET /balances placeholder (indexer/read-model will populate)
	r.Method(http.MethodGet, "/balances", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	}))
// x
	log.Printf("API listening on :%s", cfg.HTTPPort)
	if err := http.ListenAndServe(":"+cfg.HTTPPort, r); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
