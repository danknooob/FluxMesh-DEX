package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/danknooob/fluxmesh-dex/api/internal/auth"
	"github.com/danknooob/fluxmesh-dex/api/internal/config"
	"github.com/danknooob/fluxmesh-dex/api/internal/models"
	"github.com/danknooob/fluxmesh-dex/api/internal/service"
)

type AuthController struct {
	cfg     *config.Config
	userSvc service.UserService
}

func NewAuthController(cfg *config.Config, userSvc service.UserService) *AuthController {
	return &AuthController{cfg: cfg, userSvc: userSvc}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type authResponse struct {
	AccessToken string `json:"access_token"`
	Role        string `json:"role"`
	UserID      string `json:"user_id"`
}

// Login authenticates against Postgres via bcrypt and issues a JWT.
func (c *AuthController) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" {
		http.Error(w, "email and password are required", http.StatusBadRequest)
		return
	}

	user, err := c.userSvc.Authenticate(req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			http.Error(w, "invalid email or password", http.StatusUnauthorized)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	token, err := auth.NewToken(user.ID.String(), string(user.Role), c.cfg)
	if err != nil {
		http.Error(w, "could not issue token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(authResponse{
		AccessToken: token,
		Role:        string(user.Role),
		UserID:      user.ID.String(),
	})
}

// Register creates a new user with a bcrypt-hashed password.
func (c *AuthController) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" {
		http.Error(w, "email and password are required", http.StatusBadRequest)
		return
	}
	if len(req.Password) < 6 {
		http.Error(w, "password must be at least 6 characters", http.StatusBadRequest)
		return
	}

	role := models.UserRoleTrader
	if req.Role == "admin" {
		role = models.UserRoleAdmin
	}

	user, err := c.userSvc.Register(req.Email, req.Password, role)
	if err != nil {
		if errors.Is(err, service.ErrEmailTaken) {
			http.Error(w, "email already registered", http.StatusConflict)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	token, err := auth.NewToken(user.ID.String(), string(user.Role), c.cfg)
	if err != nil {
		http.Error(w, "could not issue token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(authResponse{
		AccessToken: token,
		Role:        string(user.Role),
		UserID:      user.ID.String(),
	})
}
