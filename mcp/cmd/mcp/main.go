package main

import (
	"log"
	"net/http"

	"github.com/danknooob/fluxmesh-dex/mcp/internal/config"
	"github.com/danknooob/fluxmesh-dex/mcp/internal/handler"
)

func main() {
	cfg := config.Load()
	admin := &handler.AdminHandler{}

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

	log.Printf("Control plane API listening on :%s", cfg.HTTPPort)
	if err := http.ListenAndServe(":"+cfg.HTTPPort, mux); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
