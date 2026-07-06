package model

import (
	"time"
)

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UpdateUserRequest struct {
	FullName *string `json:"full_name,omitempty"`
	Email    *string `json:"email,omitempty"`
}
