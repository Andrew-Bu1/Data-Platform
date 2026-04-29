package handler

type CreatePipelineRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type PipelineResponse struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}
