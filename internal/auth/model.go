package auth

import (
	"errors"
	"time"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailExists        = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidRefresh     = errors.New("invalid or expired refresh token")
	ErrInvalidToken = errors.New("invalid or expired token")
 ErrTokenReplay = errors.New("refresh token replay detected")
	ErrInvalidInput = errors.New("invalid input")
)

type Handler struct { Service *Service}

type User struct {
ID uint64 `json:"id"`
Name string `json:"name"`
Email string `json:"email"`
PasswordHash string `json:"-"`
CreatedAt time.Time `json:"createdat"`
UpdatedAt time.Time `json:"updtedat"`
}

type PasswordToken struct {
	AccessToken string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType string `json:"token_type"`
	ExpiresIn int64 `json:"expires_in"`
}

type AuthResponse struct {
	User User `json:"user"`
	Tokens PasswordToken `json:"tokens"`
}

type RegisterRequest struct {
	Name string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginRequest struct {
	Email string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}


type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}


