package domain

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type Pipeline struct {
	ID          pgtype.UUID
	ProjectID   pgtype.UUID
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}