package model

import "time"

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateUserRequest struct {
	ID           string
	Email        string
	FullName     string
	PasswordHash string
}

type UpdateUserRequest struct {
	FullName *string
	Email    *string
}
