package handler

import (
	"net/http"

	"encoding/json"

	"github.com/Andrew-Bu1/api/internal/model"
	"github.com/Andrew-Bu1/api/internal/service"
)

type UserHandler struct {
	user *service.UserService
}

func NewUserHandler(user *service.UserService) *UserHandler {
	return &UserHandler{user: user}
}

func (h *UserHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /users/{id}", h.GetByID)
	mux.HandleFunc("PATCH /users/{id}", h.Update)
	mux.HandleFunc("DELETE /users/{id}", h.Delete)
}

func (h *UserHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID, err := parseUUID(id)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid UUID format")
		return
	}
	user, err := h.user.GetByID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	userID, err := parseUUID(id)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid UUID format")
		return
	}
	var req model.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.user.Update(r.Context(), userID, &req); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	userID, err := parseUUID(id)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid UUID format")
		return
	}
	if err := h.user.Delete(r.Context(), userID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nil)
}
