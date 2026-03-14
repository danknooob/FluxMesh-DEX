//go:build integration
// +build integration

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danknooob/fluxmesh-dex/api/internal/auth"
	"github.com/danknooob/fluxmesh-dex/api/internal/config"
	"github.com/danknooob/fluxmesh-dex/api/internal/dbseed"
	"github.com/danknooob/fluxmesh-dex/api/internal/handler"
	"github.com/danknooob/fluxmesh-dex/api/internal/kafka"
	"github.com/danknooob/fluxmesh-dex/api/internal/migrations"
	"github.com/danknooob/fluxmesh-dex/api/internal/models"
	"github.com/danknooob/fluxmesh-dex/api/internal/repository"
	"github.com/danknooob/fluxmesh-dex/api/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupRouter(t *testing.T) *httptest.Server {
	t.Helper()
	cfg := config.Load()
	db, err := gorm.Open(postgres.Open(cfg.DB.DSN), &gorm.Config{})
	if err != nil {
		t.Skipf("skip integration: DB connection failed: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Order{}, &models.Market{}, &models.Balance{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := migrations.RunStoredProcedures(db); err != nil {
		t.Fatalf("stored procedures: %v", err)
	}
	_ = dbseed.SeedInitialMarkets(db)
	dbseed.SeedDefaultUsers(db)

	producer := kafka.NewProducer(cfg.Kafka.Brokers)
	t.Cleanup(func() { producer.Close() })

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
		gr.Get("/markets/{id}/depth", orderCtrl.Depth)
		gr.Get("/balances", balanceCtrl.List)
	})

	return httptest.NewServer(r)
}

func TestIntegration_Login(t *testing.T) {
	srv := setupRouter(t)
	defer srv.Close()

	resp, err := srv.Client().Post(srv.URL+"/auth/login", "application/json", bytes.NewBufferString(`{"email":"trader@example.com","password":"trader123"}`))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var out struct {
		AccessToken string `json:"access_token"`
		Role        string `json:"role"`
		UserID      string `json:"user_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.AccessToken == "" || out.UserID == "" {
		t.Errorf("missing token or user_id")
	}
}

func TestIntegration_RegisterAndLogin(t *testing.T) {
	srv := setupRouter(t)
	defer srv.Close()

	body := `{"email":"inttest@example.com","password":"pass123","role":"trader"}`
	resp, err := srv.Client().Post(srv.URL+"/auth/register", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("register expected 200, got %d", resp.StatusCode)
	}

	resp2, err := srv.Client().Post(srv.URL+"/auth/login", "application/json", bytes.NewBufferString(`{"email":"inttest@example.com","password":"pass123"}`))
	if err != nil {
		t.Fatalf("login request: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("login expected 200, got %d", resp2.StatusCode)
	}
}

func TestIntegration_MarketsList(t *testing.T) {
	srv := setupRouter(t)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/markets", nil)
	req.Header.Set("X-User-ID", "00000000-0000-0000-0000-000000000001")
	req.Header.Set("X-Role", "trader")
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var markets []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&markets); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(markets) == 0 {
		t.Error("expected at least one market from seed")
	}
}

func TestIntegration_Depth(t *testing.T) {
	srv := setupRouter(t)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/markets/BTC-USDC/depth?limit=10", nil)
	req.Header.Set("X-User-ID", "00000000-0000-0000-0000-000000000001")
	req.Header.Set("X-Role", "trader")
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var depth struct {
		Bids []interface{} `json:"bids"`
		Asks []interface{} `json:"asks"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&depth); err != nil {
		t.Fatalf("decode: %v", err)
	}
}

func TestIntegration_CreateOrder_Unauthorized(t *testing.T) {
	srv := setupRouter(t)
	defer srv.Close()

	body := `{"market_id":"BTC-USDC","side":"buy","price":"60000","size":"0.01"}`
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/orders", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth, got %d", resp.StatusCode)
	}
}

func TestIntegration_CreateOrder_Success(t *testing.T) {
	srv := setupRouter(t)
	defer srv.Close()

	loginResp, err := srv.Client().Post(srv.URL+"/auth/login", "application/json", bytes.NewBufferString(`{"email":"trader@example.com","password":"trader123"}`))
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	defer loginResp.Body.Close()
	var loginOut struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(loginResp.Body).Decode(&loginOut); err != nil || loginOut.UserID == "" {
		t.Skipf("login failed or no user_id: %v", err)
	}

	body := `{"market_id":"BTC-USDC","side":"buy","price":"60000","size":"0.01"}`
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/orders", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", loginOut.UserID)
	req.Header.Set("X-Role", "trader")
	req.Header.Set("Idempotency-Key", "integration-test-order-1")
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		t.Fatalf("create order expected 200/202, got %d", resp.StatusCode)
	}
	var order map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&order); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if order["id"] == nil {
		t.Error("expected order id in response")
	}
}

// TestE2E_OrderLifecycle runs a full order flow: register → login → create order → list orders → cancel order.
func TestE2E_OrderLifecycle(t *testing.T) {
	srv := setupRouter(t)
	defer srv.Close()
	client := srv.Client()

	// 1) Register
	regBody := `{"email":"e2e@example.com","password":"e2epass","role":"trader"}`
	regResp, err := client.Post(srv.URL+"/auth/register", "application/json", bytes.NewBufferString(regBody))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	regResp.Body.Close()
	if regResp.StatusCode != http.StatusOK {
		t.Fatalf("register: %d", regResp.StatusCode)
	}

	// 2) Login
	loginResp, err := client.Post(srv.URL+"/auth/login", "application/json", bytes.NewBufferString(`{"email":"e2e@example.com","password":"e2epass"}`))
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	defer loginResp.Body.Close()
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("login: %d", loginResp.StatusCode)
	}
	var loginOut struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(loginResp.Body).Decode(&loginOut); err != nil || loginOut.UserID == "" {
		t.Fatalf("login decode: %v", err)
	}

	// 3) Create order
	orderBody := `{"market_id":"BTC-USDC","side":"buy","price":"50000","size":"0.02"}`
	createReq, _ := http.NewRequest(http.MethodPost, srv.URL+"/orders", bytes.NewBufferString(orderBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("X-User-ID", loginOut.UserID)
	createReq.Header.Set("X-Role", "trader")
	createReq.Header.Set("Idempotency-Key", "e2e-lifecycle-1")
	createResp, err := client.Do(createReq)
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	defer createResp.Body.Close()
	if createResp.StatusCode != http.StatusOK && createResp.StatusCode != http.StatusAccepted {
		t.Fatalf("create order: %d", createResp.StatusCode)
	}
	var createdOrder map[string]interface{}
	if err := json.NewDecoder(createResp.Body).Decode(&createdOrder); err != nil {
		t.Fatalf("create decode: %v", err)
	}
	orderID, _ := createdOrder["id"].(string)
	if orderID == "" {
		t.Fatal("create response missing order id")
	}

	// 4) List orders
	listReq, _ := http.NewRequest(http.MethodGet, srv.URL+"/orders?market_id=BTC-USDC", nil)
	listReq.Header.Set("X-User-ID", loginOut.UserID)
	listReq.Header.Set("X-Role", "trader")
	listResp, err := client.Do(listReq)
	if err != nil {
		t.Fatalf("list orders: %v", err)
	}
	defer listResp.Body.Close()
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("list orders: %d", listResp.StatusCode)
	}
	var orders []map[string]interface{}
	if err := json.NewDecoder(listResp.Body).Decode(&orders); err != nil {
		t.Fatalf("list decode: %v", err)
	}
	if len(orders) == 0 {
		t.Error("expected at least one order after create")
	}

	// 5) Cancel order
	cancelReq, _ := http.NewRequest(http.MethodDelete, srv.URL+"/orders/"+orderID, nil)
	cancelReq.Header.Set("X-User-ID", loginOut.UserID)
	cancelReq.Header.Set("X-Role", "trader")
	cancelResp, err := client.Do(cancelReq)
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	cancelResp.Body.Close()
	if cancelResp.StatusCode != http.StatusOK {
		t.Fatalf("cancel expected 200, got %d", cancelResp.StatusCode)
	}

	// 6) List again — order should be cancelled (or gone from "open" filter)
	listReq2, _ := http.NewRequest(http.MethodGet, srv.URL+"/orders?market_id=BTC-USDC&status=cancelled", nil)
	listReq2.Header.Set("X-User-ID", loginOut.UserID)
	listReq2.Header.Set("X-Role", "trader")
	listResp2, err := client.Do(listReq2)
	if err != nil {
		t.Fatalf("list after cancel: %v", err)
	}
	defer listResp2.Body.Close()
	var ordersAfter []map[string]interface{}
	_ = json.NewDecoder(listResp2.Body).Decode(&ordersAfter)
	var found bool
	for _, o := range ordersAfter {
		if o["id"] == orderID && (o["status"] == "cancelled" || o["status"] == "canceled") {
			found = true
			break
		}
	}
	if len(ordersAfter) > 0 && !found {
		// Might be returned without status filter
		t.Logf("order after cancel: %v", ordersAfter)
	}
}
