package handler

import (
	"encoding/json"
	"net/http"

	"github.com/danknooob/fluxmesh-dex/api/internal/auth"
	"github.com/danknooob/fluxmesh-dex/api/internal/config"
)

// AuthController handles basic login for issuing JWTs.
type AuthController struct {
	cfg *config.Config
}

// NewAuthController creates an AuthController.
func NewAuthController(cfg *config.Config) *AuthController {
	return &AuthController{cfg: cfg}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken string `json:"access_token"`
	Role        string `json:"role"`
}

// Login issues a JWT for a valid user (dev only: hard-coded users).
func (c *AuthController) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	var userID, role string
	switch {
	case req.Email == "admin@example.com" && req.Password == "admin":
		userID, role = "admin-1", "admin"
	case req.Email == "trader@example.com" && req.Password == "trader":
		userID, role = "trader-1", "trader"
	default:
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := auth.NewToken(userID, role, c.cfg)
	if err != nil {
		http.Error(w, "could not issue token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(loginResponse{
		AccessToken: token,
		Role:        role,
	})
}

