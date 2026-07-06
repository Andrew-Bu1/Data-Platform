package service

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/Andrew-Bu1/api/internal/model"
	"github.com/Andrew-Bu1/api/internal/repository"
	"github.com/Andrew-Bu1/api/internal/utils"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	authRepo  *repository.AuthRepository
	log       *slog.Logger
	jwtSecret string
}

func NewAuthService(authRepo *repository.AuthRepository, log *slog.Logger, jwtSecret string) *AuthService {
	return &AuthService{
		authRepo:  authRepo,
		log:       log,
		jwtSecret: jwtSecret,
	}
}

func (s *AuthService) Register(ctx context.Context, req *model.RegisterRequest) error {
	s.log.Info("user registering", slog.String("email", req.Email))

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.log.Error("failed to generate password hash", slog.String("email", req.Email), slog.Any("error", err))
		return err
	}
	user := &model.CreateUserParams{
		ID:           uuid.New(),
		Email:        req.Email,
		FullName:     strings.Split(req.Email, "@")[0],
		PasswordHash: string(passwordHash),
	}
	if err := s.authRepo.CreateUser(ctx, user); err != nil {
		s.log.Error("failed to register user", slog.String("email", req.Email), slog.Any("error", err))
		return err
	}

	return nil
}

func (s *AuthService) Login(ctx context.Context, req *model.LoginRequest) (*model.AuthResponse, error) {
	user, err := s.authRepo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		s.log.Error("failed to get user by email", slog.String("email", req.Email), slog.Any("error", err))
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		s.log.Error("failed to compare password hash", slog.String("email", req.Email), slog.Any("error", err))
		return nil, err
	}

	// Create access token and refresh token
	accessToken, err := utils.GenerateAccessToken(user.ID.String(), user.Email, s.jwtSecret)
	if err != nil {
		s.log.Error("failed to generate access token", slog.String("email", req.Email), slog.Any("error", err))
		return nil, err
	}

	refreshToken, err := utils.GenerateRandomToken()
	if err != nil {
		s.log.Error("failed to generate refresh token", slog.String("email", req.Email), slog.Any("error", err))
		return nil, err
	}
	err = s.authRepo.CreateSession(ctx, &model.CreateSessionParams{
		ID:               uuid.New(),
		UserID:           user.ID,
		SessionTokenHash: utils.HashToken(refreshToken),
		ExpiresAt:        time.Now().Add(24 * time.Hour),
	})
	if err != nil {
		s.log.Error("failed to create session", slog.String("email", req.Email), slog.Any("error", err))
		return nil, err
	}

	return &model.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, req *model.LogoutRequest) error {
	tokenHash := utils.HashToken(req.RefreshToken)
	session, err := s.authRepo.GetActiveSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		return err
	}

	if err := s.authRepo.RevokeSession(ctx, session.ID); err != nil {
		s.log.Error("failed to revoke session", slog.Any("error", err))
		return err
	}

	return nil
}
