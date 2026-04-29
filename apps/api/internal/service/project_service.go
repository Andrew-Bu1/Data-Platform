package service

import (
	"context"
	"fmt"

	"data-platform/api/internal/domain"
	"data-platform/api/internal/repository"
)

type ProjectService struct {
	repo *repository.ProjectRepository
}

func NewProjectService(repo *repository.ProjectRepository) *ProjectService {
	return &ProjectService{repo: repo}
}

func (s *ProjectService) Create(ctx context.Context, p *domain.Project) error {
	if p.Name == "" {
		return fmt.Errorf("project name is required")
	}
	return s.repo.Create(ctx, p)
}

func (s *ProjectService) GetByID(ctx context.Context, id string) (*domain.Project, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("project %s not found", id)
	}
	return p, nil
}

func (s *ProjectService) List(ctx context.Context) ([]*domain.Project, error) {
	return s.repo.List(ctx)
}

func (s *ProjectService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
