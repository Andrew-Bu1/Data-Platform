package main

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/Andrew-Bu1/api/internal/config"
	"github.com/Andrew-Bu1/api/internal/handler"
	"github.com/Andrew-Bu1/api/internal/logger"
	"github.com/Andrew-Bu1/api/internal/repository"
	"github.com/Andrew-Bu1/api/internal/service"
)

func main() {
	mux := http.NewServeMux()

	cfg := config.Load()
	logger := logger.New(cfg.Env)

	slog.SetDefault(logger)

	slog.Info("server starting")

	ctx := context.Background()
	// Setup database
	pool, err := repository.NewPool(ctx, cfg.DatabaseURL, logger)
	if err != nil {
		slog.Error("database connection failed", slog.Any("error", err))
		return
	}
	defer pool.Close()

	// Setup Repositories
	userRepo := repository.NewUserRepository(pool)

	// Setup Services
	userService := service.NewUserService(userRepo, logger)

	// Setup Handlers
	userHandler := handler.NewUserHandler(userService)

	userHandler.RegisterRoutes(mux)

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	if err := server.ListenAndServe(); err != nil {
		slog.Error("server failed", slog.Any("error", err))
	}
}
