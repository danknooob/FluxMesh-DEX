package main

import (
	"log"
	"net/http"

	"github.com/danknooob/fluxmesh-dex/mcp/internal/config"
	"github.com/danknooob/fluxmesh-dex/mcp/internal/handler"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()

	var db *gorm.DB
	if cfg.DB != "" {
		if d, err := gorm.Open(postgres.Open(cfg.DB), &gorm.Config{}); err == nil {
			db = d
			log.Println("control-plane: connected to Postgres for config")
		} else {
			log.Printf("control-plane: failed to connect Postgres (config will be empty): %v", err)
		}
	}

	admin := &handler.AdminHandler{DB: db}

	mux := http.NewServeMux()
	mux.HandleFunc("/admin/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			admin.Config(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})
	mux.HandleFunc("/admin/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			admin.Health(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})
	mux.HandleFunc("/admin/markets", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			admin.Markets(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})

	log.Printf("Control plane API listening on :%s", cfg.HTTPPort)
	if err := http.ListenAndServe(":"+cfg.HTTPPort, mux); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
