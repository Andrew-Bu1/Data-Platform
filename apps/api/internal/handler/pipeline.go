package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"data-platform/api/internal/domain"
	"data-platform/api/internal/service"

	"github.com/jackc/pgx/v5/pgtype"
)

// Routes are nested under /projects/{project_id}/pipelines

type PipelineHandler struct {
	svc *service.PipelineService
}

func NewPipelineHandler(svc *service.PipelineService) *PipelineHandler {
	return &PipelineHandler{svc: svc}
}

func (h *PipelineHandler) Create(w http.ResponseWriter, r *http.Request) {
	projectIDStr := r.PathValue("project_id")

	var projectID pgtype.UUID
	if err := projectID.Scan(projectIDStr); err != nil {
		http.Error(w, "invalid project_id", http.StatusBadRequest)
		return
	}

	var req CreatePipelineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	p := &domain.Pipeline{
		ProjectID:   projectID,
		Name:        req.Name,
		Description: req.Description,
	}

	if err := h.svc.Create(r.Context(), p); err != nil {
		slog.Error("failed to create pipeline", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(PipelineResponse{
		ID:          p.ID.String(),
		ProjectID:   p.ProjectID.String(),
		Name:        p.Name,
		Description: p.Description,
		CreatedAt:   formatTime(p.CreatedAt),
		UpdatedAt:   formatTime(p.UpdatedAt),
	})
}

func (h *PipelineHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	p, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		slog.Warn("pipeline not found", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(PipelineResponse{
		ID:          p.ID.String(),
		ProjectID:   p.ProjectID.String(),
		Name:        p.Name,
		Description: p.Description,
		CreatedAt:   formatTime(p.CreatedAt),
		UpdatedAt:   formatTime(p.UpdatedAt),
	})
}

func (h *PipelineHandler) List(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("project_id")
	if projectID == "" {
		http.Error(w, "project_id is required", http.StatusBadRequest)
		return
	}

	pipelines, err := h.svc.List(r.Context(), projectID)
	if err != nil {
		slog.Error("failed to list pipelines", "project_id", projectID, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := make([]PipelineResponse, len(pipelines))
	for i, p := range pipelines {
		resp[i] = PipelineResponse{
			ID:          p.ID.String(),
			ProjectID:   p.ProjectID.String(),
			Name:        p.Name,
			Description: p.Description,
			CreatedAt:   formatTime(p.CreatedAt),
			UpdatedAt:   formatTime(p.UpdatedAt),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *PipelineHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.svc.Delete(r.Context(), id); err != nil {
		slog.Error("failed to delete pipeline", "id", id, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
