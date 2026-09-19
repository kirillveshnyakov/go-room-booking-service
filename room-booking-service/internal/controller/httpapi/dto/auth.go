package dto

import "time"

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RegisterResponse struct {
	User UserResponse `json:"user"`
}

type UserResponse struct {
	ID        string     `json:"id"`
	Email     string     `json:"email"`
	Role      string     `json:"role"`
	CreatedAt *time.Time `json:"createdAt,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type TokenResponse struct {
	Token string `json:"token"`
}

type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
}

type DummyLoginRequest struct {
	Role string `json:"role" binding:"required,oneof=admin user"`
}
