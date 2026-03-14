package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/danknooob/fluxmesh-dex/api/internal/auth"
	"github.com/danknooob/fluxmesh-dex/api/internal/service"
	"github.com/google/uuid"
)

type ProfileController struct {
	userSvc service.UserService
}

func NewProfileController(userSvc service.UserService) *ProfileController {
	return &ProfileController{userSvc: userSvc}
}

func (c *ProfileController) Get(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(auth.UserIDFrom(r.Context()))
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := c.userSvc.GetProfile(r.Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(user)
}

func (c *ProfileController) Update(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(auth.UserIDFrom(r.Context()))
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req service.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	user, err := c.userSvc.UpdateProfile(r.Context(), userID, req)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, service.ErrEmailTaken) {
			http.Error(w, "email already taken", http.StatusConflict)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(user)
}

func (c *ProfileController) Delete(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(auth.UserIDFrom(r.Context()))
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := c.userSvc.DeleteAccount(r.Context(), userID); err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
