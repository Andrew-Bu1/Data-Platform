package handler

import (
	"encoding/json"
	"net/http"

	"github.com/Andrew-Bu1/api/internal/model"
	"github.com/Andrew-Bu1/api/internal/service"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("POST /auth/register", http.HandlerFunc(h.Register))
	mux.Handle("POST /auth/login", http.HandlerFunc(h.Login))
	mux.Handle("POST /auth/logout", http.HandlerFunc(h.Logout))
}

// Register godoc
// @Summary Register user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body model.RegisterRequest true "Register request"
// @Success 201
// @Failure 400 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	if err := h.auth.Register(r.Context(), &req); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to register user")
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// Login godoc
// @Summary Login user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body model.LoginRequest true "Login request"
// @Success 200 {object} model.AuthResponseEnvelope
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Router /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	token, err := h.auth.Login(r.Context(), &req)

	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	writeJSON(w, http.StatusOK, token)
}

// Logout godoc
// @Summary Logout user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body model.LogoutRequest true "Logout request"
// @Success 200
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req model.LogoutRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.auth.Logout(r.Context(), &req); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid session")
		return
	}

	w.WriteHeader(http.StatusOK)
}
