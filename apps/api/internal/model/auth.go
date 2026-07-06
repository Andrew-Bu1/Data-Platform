package model

type RegisterRequest struct {
	Email    string `string:"email"`
	Fullname string `string:"full_name"`
	password string `string:"password"`
}

type LoginRequest struct {
	Email    string `string:"email"`
	password string `string:"email"`
}
