package repository

import (
	"context"

	"github.com/Andrew-Bu1/api/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	query := `
		SELECT id, email, full_name, created_at, updated_at 
		from users
		WHERE id =  $1
	`
	var user model.User

	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.FullName,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) Update(ctx context.Context, id uuid.UUID, req *model.UpdateUserRequest) error {
	query := `
		UPDATE users
		SET full_name = COALESCE($2, full_name),
		email = COALESCE($3, email),
		updated_at = now()
		WHERE id = $1
	`

	tag, err := r.db.Exec(ctx, query, id, req.FullName, req.Email)

	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM users
		WHERE id = $1
	`

	tag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}
