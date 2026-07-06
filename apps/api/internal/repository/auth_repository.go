package repository

import (
	"context"

	"github.com/Andrew-Bu1/api/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthRepository struct {
	db *pgxpool.Pool
}

func NewAuthRepository(db *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{db: db}
}

func (r *AuthRepository) CreateUser(ctx context.Context, params *model.CreateUserParams) error {
	query := `
		INSERT INTO users (id, email, full_name, password_hash)
		VALUES ($1, $2, $3, $4)
	`

	_, err := r.db.Exec(ctx, query, params.ID, params.Email, params.FullName, params.PasswordHash)
	if err != nil {
		return err
	}

	return nil
}

func (r *AuthRepository) GetUserByEmail(ctx context.Context, email string) (*model.AuthUser, error) {
	query := `
		SELECT id, email, password_hash
		from users
		WHERE email = $1
	`
	var user model.AuthUser

	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *AuthRepository) CreateSession(ctx context.Context, params *model.CreateSessionParams) error {
	query := `
		INSERT INTO sessions (id, user_id, session_token_hash, expires_at)
		VALUES ($1, $2, $3, $4)
	`

	_, err := r.db.Exec(ctx, query, params.ID, params.UserID, params.SessionTokenHash, params.ExpiresAt)
	if err != nil {
		return err
	}

	return nil
}

func (r *AuthRepository) GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (*model.Session, error) {
	query := `
		SELECT id, user_id, session_token_hash, created_at, revoked_at, expires_at
		FROM sessions
		WHERE session_token_hash = $1 
			AND revoked_at IS NULL
			AND expires_at > now()
	`
	var session model.Session

	err := r.db.QueryRow(ctx, query, tokenHash).Scan(
		&session.ID,
		&session.UserID,
		&session.SessionTokenHash,
		&session.CreatedAt,
		&session.RevokedAt,
		&session.ExpiresAt,
	)

	if err != nil {
		return nil, err
	}

	return &session, nil
}

func (r *AuthRepository) RevokeSession(ctx context.Context, sessionID uuid.UUID) error {
	query := `
		UPDATE sessions
		SET revoked_at = now()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, sessionID)
	return err
}
