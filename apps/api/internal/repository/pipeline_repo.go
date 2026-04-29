package repository

import (
	"context"
	"errors"

	"data-platform/api/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PipelineRepository struct {
	db *pgxpool.Pool
}

func NewPipelineRepository(db *pgxpool.Pool) *PipelineRepository {
	return &PipelineRepository{db: db}
}

func (r *PipelineRepository) Create(ctx context.Context, p *domain.Pipeline) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO pipelines (id, project_id, name, description, created_at, updated_at)
		 VALUES (gen_random_uuid(), $1, $2, $3, now(), now())`,
		p.ProjectID, p.Name, p.Description,
	)
	return err
}

func (r *PipelineRepository) GetByID(ctx context.Context, id string) (*domain.Pipeline, error) {
	p := &domain.Pipeline{}
	err := r.db.QueryRow(ctx,
		`SELECT id, project_id, name, description, created_at, updated_at
		 FROM pipelines WHERE id = $1`,
		id,
	).Scan(&p.ID, &p.ProjectID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *PipelineRepository) List(ctx context.Context, projectID string) ([]*domain.Pipeline, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, project_id, name, description, created_at, updated_at
		 FROM pipelines WHERE project_id = $1 ORDER BY created_at DESC`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pipelines []*domain.Pipeline
	for rows.Next() {
		p := &domain.Pipeline{}
		if err := rows.Scan(&p.ID, &p.ProjectID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		pipelines = append(pipelines, p)
	}
	return pipelines, rows.Err()
}

func (r *PipelineRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM pipelines WHERE id = $1`, id)
	return err
}