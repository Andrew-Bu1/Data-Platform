package service

import (
	"context"
	"fmt"

	"data-platform/api/internal/domain"
	"data-platform/api/internal/repository"
)

type PipelineService struct {
	repo *repository.PipelineRepository
}

func NewPipelineService(repo *repository.PipelineRepository) *PipelineService {
	return &PipelineService{repo: repo}
}

func (s *PipelineService) Create(ctx context.Context, p *domain.Pipeline) error {
	if p.Name == "" {
		return fmt.Errorf("pipeline name is required")
	}
	return s.repo.Create(ctx, p)
}

func (s *PipelineService) GetByID(ctx context.Context, id string) (*domain.Pipeline, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("pipeline %s not found", id)
	}
	return p, nil
}

func (s *PipelineService) List(ctx context.Context, projectID string) ([]*domain.Pipeline, error) {
	return s.repo.List(ctx, projectID)
}

func (s *PipelineService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
