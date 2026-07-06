package main

import (
	"context"
	"log/slog"
	"net/http"

	_ "github.com/Andrew-Bu1/api/docs"
	"github.com/Andrew-Bu1/api/internal/config"
	"github.com/Andrew-Bu1/api/internal/handler"
	"github.com/Andrew-Bu1/api/internal/logger"
	"github.com/Andrew-Bu1/api/internal/middleware"
	"github.com/Andrew-Bu1/api/internal/repository"
	"github.com/Andrew-Bu1/api/internal/service"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// @title Data Platform API
// @version 0.1
// @description Data Platform backend API.
// @host localhost:8080
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
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
	authRepo := repository.NewAuthRepository(pool)
	userRepo := repository.NewUserRepository(pool)

	// Setup Services
	authService := service.NewAuthService(authRepo, logger, cfg.JwtSecret)
	userService := service.NewUserService(userRepo, logger)

	// Setup Middleware
	authMiddleware := middleware.NewAuthMiddleware(logger, cfg.JwtSecret)

	// Setup Handlers
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)

	userHandler.RegisterRoutes(mux, authMiddleware.RequireAuth)
	authHandler.RegisterRoutes(mux)

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("GET /swagger/", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"),
	))

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	if err := server.ListenAndServe(); err != nil {
		slog.Error("server failed", slog.Any("error", err))
	}
}
