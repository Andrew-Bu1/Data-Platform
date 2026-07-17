package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Andrew-Bu1/api/internal/middleware"
	"github.com/Andrew-Bu1/api/internal/model"
	"github.com/Andrew-Bu1/api/internal/repository"
	"github.com/Andrew-Bu1/api/internal/service"
)

type UserHandler struct {
	user *service.UserService
}

func NewUserHandler(user *service.UserService) *UserHandler {
	return &UserHandler{user: user}
}

func (h *UserHandler) RegisterRoutes(mux *http.ServeMux, requireAuth func(http.Handler) http.Handler) {
	mux.Handle("GET /users/{id}", requireAuth(http.HandlerFunc(h.GetByID)))
	mux.Handle("PATCH /users/{id}", requireAuth(http.HandlerFunc(h.Update)))
	mux.Handle("DELETE /users/{id}", requireAuth(http.HandlerFunc(h.Delete)))
}

// GetByID godoc
// @Summary Get user by ID
// @Tags users
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} model.Response[model.User]
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /users/{id} [get]
func (h *UserHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID, err := parseUUID(id)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid UUID format")
		return
	}
	user, err := h.user.GetByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, user)
}

// Update godoc
// @Summary Update user
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param request body model.UpdateUserRequest true "Update user request"
// @Success 204 {object} nil
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /users/{id} [patch]
func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	userID, err := parseUUID(id)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid UUID format")
		return
	}

	callerID, ok := middleware.GetUserID(r.Context())
	if !ok || callerID != userID.String() {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	var req model.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.user.Update(r.Context(), userID, &req); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

// Delete godoc
// @Summary Delete user
// @Tags users
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 204 {object} nil
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /users/{id} [delete]
func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	userID, err := parseUUID(id)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid UUID format")
		return
	}

	callerID, ok := middleware.GetUserID(r.Context())
	if !ok || callerID != userID.String() {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	if err := h.user.Delete(r.Context(), userID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}
