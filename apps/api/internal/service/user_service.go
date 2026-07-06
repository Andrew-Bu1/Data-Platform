package service

import (
	"context"
	"log/slog"

	"github.com/Andrew-Bu1/api/internal/model"
	"github.com/Andrew-Bu1/api/internal/repository"
	"github.com/google/uuid"
)

type UserService struct {
	userRepo *repository.UserRepository
	log      *slog.Logger
}

func NewUserService(userRepo *repository.UserRepository, log *slog.Logger) *UserService {
	return &UserService{userRepo: userRepo, log: log}
}

func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		s.log.Error("user not found", slog.String("user_id", id.String()))
		return nil, err
	}
	return user, nil
}

func (s *UserService) Update(ctx context.Context, id uuid.UUID, req *model.UpdateUserRequest) error {
	err := s.userRepo.Update(ctx, id, req)
	if err != nil {
		s.log.Error("failed to update user", slog.String("user_id", id.String()), slog.Any("error", err))
		return err
	}
	return nil
}

func (s *UserService) Delete(ctx context.Context, id uuid.UUID) error {
	err := s.userRepo.Delete(ctx, id)
	if err != nil {
		s.log.Error("failed to delete user", slog.String("user_id", id.String()), slog.Any("error", err))
		return err
	}
	return nil
}
