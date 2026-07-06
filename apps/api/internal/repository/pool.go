package repository

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a requested record does not exist.
var ErrNotFound = errors.New("not found")

func NewPool(ctx context.Context, databaseURL string, logger *slog.Logger) (*pgxpool.Pool, error) {
	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(ctx); err != nil {
		return nil, err
	}

	logger.Info("connected to database")
	return db, nil
}
