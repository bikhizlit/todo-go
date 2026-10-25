
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"
)

 const(
 accessTokenTTL = 15 * time.Minute
	refreshTokenTTL = 30 * 24 * time.Hour
)

type Service struct {
	repo   *UserRepository
	tokens *AccessTokenManager
}

func NewService(
	repo *UserRepository,
	tokens *AccessTokenManager,
) *Service {
	return &Service{
		repo:   repo,
		tokens: tokens,
	}
}

func normalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	if len(email) == 0 || len(email) > 255 {
		return "", fmt.Errorf("email must contain 1 to 255 bytes")
	}

	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return "", fmt.Errorf("invalid email address")
	}

	return email, nil
}

func validateName(name string) (string, error) {
	name = strings.TrimSpace(name)

	if !utf8.ValidString(name) ||
		utf8.RuneCountInString(name) < 1 ||
		utf8.RuneCountInString(name) > 100 {
		return "", fmt.Errorf("name must contain 1 to 100 characters")
	}

	return name, nil
}

func newRefreshToken() (string, error) {
	randomBytes := make([]byte, 32)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(randomBytes), nil
}

func (s *Service) issueTokens(
	ctx context.Context,
	userID uint64,
 ) (PasswordToken, error) {
	accessToken, err := s.tokens.Issue(userID)
	if err != nil {
		return PasswordToken{}, err
	}

	refreshToken, err := newRefreshToken()
	if err != nil {
		return PasswordToken{}, err
	}

	if err := s.repo.CreateRefreshSession(
		ctx,
		userID,
		refreshToken,
		time.Now().UTC().Add(refreshTokenTTL),
	); err != nil {
		return PasswordToken{}, err
	}

	return PasswordToken{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(accessTokenTTL.Seconds()),
	}, nil
}

func (s *Service) Register(
	ctx context.Context,
	name, email, password string,
) (*AuthResponse, error) {
	name, err := validateName(name)
	if err != nil {
		return nil, err
	}

	email, err = normalizeEmail(email)
	if err != nil {
		return nil, err
	}

	passwordHash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	_, err = s.repo.FindByEmail(ctx, email)
	switch {
	case err == nil:
		return nil, ErrEmailExists
	case !errors.Is(err, ErrUserNotFound):
		return nil, err
	}

	user, err := s.repo.CreateUser(ctx, name, email, passwordHash)
	if err != nil {
		return nil, err
	}

	tokens, err := s.issueTokens(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		User:   *user,
		Tokens: tokens,
	}, nil
}

func (s *Service) Login(
	ctx context.Context,
	email, password string,
) (*AuthResponse, error) {
	email, err := normalizeEmail(email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	valid, err := VerifyPassword(password, user.PasswordHash)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, ErrInvalidCredentials
	}

	tokens, err := s.issueTokens(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		User:   *user,
		Tokens: tokens,
	}, nil
}

func (s *Service) Refresh(
	ctx context.Context,
	oldRefreshToken string,
) (*AuthResponse, error) {
	if len(oldRefreshToken) == 0 || len(oldRefreshToken) > 256 {
		return nil, ErrInvalidRefresh
	}

	newRefreshToken, err := newRefreshToken()
	if err != nil {
		return nil, err
	}

	// Generate the access token before consuming the old refresh token.
	// This avoids rotating the session if token signing fails.
	//
	// The user ID is obtained from the locked refresh session below,
	// so first we need a transaction that returns that ID and rotates
	// the token. See the implementation note following this file.
	_ = newRefreshToken

	return nil, fmt.Errorf("Refresh requires RotateRefreshSessionAndGetUser implementation")
}

func (s *Service) Logout(
	ctx context.Context,
	refreshToken string,
) error {
	if refreshToken == "" || len(refreshToken) > 256 {
		return ErrInvalidRefresh
	}

	return s.repo.RevokeRefreshSession(ctx, refreshToken)
}

func (s *Service) GetUser(
	ctx context.Context,
	userID uint64,
) (*User, error) {
	return s.repo.FindById(ctx, userID)
}
