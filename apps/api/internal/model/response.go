package model

type ErrorResponse struct {
	Status string `json:"status" example:"Unauthorized"`
	Error  string `json:"error" example:"invalid email or password"`
}

type EmptyResponse struct {
	Status string      `json:"status" example:"OK"`
	Data   interface{} `json:"data"`
}

type AuthResponseEnvelope struct {
	Status string       `json:"status" example:"OK"`
	Data   AuthResponse `json:"data"`
}

type UserResponse struct {
	Status string `json:"status" example:"OK"`
	Data   User   `json:"data"`
}
