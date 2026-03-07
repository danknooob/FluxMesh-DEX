package main

import (
	"log"
	"net/http"

	"github.com/danknooob/fluxmesh-dex/api/internal/config"
	"github.com/danknooob/fluxmesh-dex/api/internal/handler"
	"github.com/danknooob/fluxmesh-dex/api/internal/kafka"
	"github.com/danknooob/fluxmesh-dex/api/internal/models"
	"github.com/danknooob/fluxmesh-dex/api/internal/repository"
	"github.com/danknooob/fluxmesh-dex/api/internal/service"
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

	producer := kafka.NewProducer(cfg.Kafka.Brokers)
	defer producer.Close()

	orderRepo := repository.NewOrderRepository(db)
	marketRepo := repository.NewMarketRepository(db)
	orderSvc := service.NewOrderService(orderRepo, marketRepo, producer)

	orderCtrl := handler.NewOrderController(orderSvc)
	marketCtrl := handler.NewMarketController(marketRepo)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", orderCtrl.Create)
	mux.HandleFunc("/orders/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			orderCtrl.Delete(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})
	mux.HandleFunc("GET /markets", marketCtrl.List)

	// GET /balances placeholder (indexer/read-model will populate)
	mux.HandleFunc("GET /balances", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	})

	log.Printf("API listening on :%s", cfg.HTTPPort)
	if err := http.ListenAndServe(":"+cfg.HTTPPort, mux); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
