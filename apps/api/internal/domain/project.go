package domain

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type Project struct {
	ID          pgtype.UUID
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
