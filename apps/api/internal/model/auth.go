package model

import (
	"time"

	"github.com/google/uuid"
)

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type CreateUserParams struct {
	ID           uuid.UUID
	Email        string
	FullName     string
	PasswordHash string
}

type CreateSessionParams struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	SessionTokenHash string
	ExpiresAt        time.Time
}

type AuthUser struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
}

type Session struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	SessionTokenHash string
	CreatedAt        time.Time
	RevokedAt        *time.Time
	ExpiresAt        time.Time
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}
