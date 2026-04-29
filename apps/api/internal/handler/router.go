package handler

import "net/http"

func NewRouter(
	projectHandler *ProjectHandler,
	pipelineHandler *PipelineHandler,
) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Projects
	mux.HandleFunc("GET /projects", projectHandler.List)
	mux.HandleFunc("POST /projects", projectHandler.Create)
	mux.HandleFunc("GET /projects/{id}", projectHandler.GetByID)
	mux.HandleFunc("DELETE /projects/{id}", projectHandler.Delete)

	// Pipelines nested under project
	mux.HandleFunc("GET /projects/{project_id}/pipelines", pipelineHandler.List)
	mux.HandleFunc("POST /projects/{project_id}/pipelines", pipelineHandler.Create)
	mux.HandleFunc("GET /projects/{project_id}/pipelines/{id}", pipelineHandler.GetByID)
	mux.HandleFunc("DELETE /projects/{project_id}/pipelines/{id}", pipelineHandler.Delete)

	return LoggingMiddleware(mux)
}
