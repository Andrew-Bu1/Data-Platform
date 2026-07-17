package model

type ErrorResponse struct {
	Status string `json:"status" example:"Unauthorized"`
	Error  string `json:"error" example:"invalid email or password"`
}

type Response[T any] struct {
	Status string `json:"status" example:"OK"`
	Data   T      `json:"data"`
}