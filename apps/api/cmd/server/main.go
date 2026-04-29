package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"data-platform/api/internal/handler"
	"data-platform/api/internal/logger"
	"data-platform/api/internal/repository"
	"data-platform/api/internal/service"
)

func main() {
	slog.SetDefault(slog.New(logger.NewColorHandler(os.Stdout)))

	if err := godotenv.Load(); err != nil {
		slog.Warn("no .env file found, using environment variables")
	}

	ctx := context.Background()

	db, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Wire dependencies
	pipelineRepo := repository.NewPipelineRepository(db)
	pipelineSvc := service.NewPipelineService(pipelineRepo)
	pipelineHandler := handler.NewPipelineHandler(pipelineSvc)

	projectRepo := repository.NewProjectRepository(db)
	projectSvc := service.NewProjectService(projectRepo)
	projectHandler := handler.NewProjectHandler(projectSvc)

	router := handler.NewRouter(projectHandler, pipelineHandler)

	addr := ":8080"
	slog.Info("server starting", "addr", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}