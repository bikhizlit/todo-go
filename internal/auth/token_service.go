package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// AccessTokenManager holds configuration parameters required for 
// issuing and validating JSON Web Tokens (JWT).
type AccessTokenManager struct {
	key      []byte        // Secret signing key used for HMAC-SHA256
	ttl      time.Duration // Token Time-To-Live (expiration duration)
	issuer   string        // Expected token issuer identifier
	audience string        // Expected token audience identifier
}

// NewTokenManager validates input configuration and initializes a new AccessTokenManager instance.
func NewTokenManager(secret []byte, issuer, audience string, ttl time.Duration) (*AccessTokenManager, error) {
	// Ensure the secret meets minimum length requirements and configuration fields are populated
	if len(secret) < 32 || issuer == "" || audience == "" || ttl <= 0 {
		return nil, errors.New("invalid JWT configuration")
	}
	
	return &AccessTokenManager{
		key:      secret,
		issuer:   issuer,
		audience: audience,
		ttl:      ttl,
	}, nil
}

// newRefreshToken generates a cryptographically secure random raw token string 
// and its corresponding SHA-256 hex-encoded digest for secure database storage.
func NewRefreshToken() (raw, digest string, err error) {
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", "", err
	}
	
	// Encode raw random bytes into a URL-safe base64 string without padding
	raw = base64.RawURLEncoding.EncodeToString(data)
	return raw, DigestRefresh(raw), nil
}

// DigestRefresh computes the SHA-256 hash of a raw refresh token and returns it as a hex string.
func DigestRefresh(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// NewFamilyID generates a Version 4 UUID formatted as a string to track session families.
func NewFamilyID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	
	// Set version to 4 (UUID v4) and variant bits according to RFC 4122
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	
	s := hex.EncodeToString(b)
	// Format the hex string into standard UUID hyphens layout
	return s[:8] + "-" + s[8:12] + "-" + s[12:16] + "-" + s[16:20] + "-" + s[20:], nil
}

// Validate parses, verifies signature and standard claims of a raw JWT string,
// and returns the corresponding user ID if valid.
func (m *AccessTokenManager) Validate(raw string) (uint64, error) {
	claims := &jwt.RegisteredClaims{}
	
	// Parse the token using the secret key and enforce security validations
	token, err := jwt.ParseWithClaims(raw, claims,
		func(_ *jwt.Token) (any, error) { return m.key, nil },
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer(m.issuer), 
		jwt.WithAudience(m.audience),
		jwt.WithExpirationRequired(), 
		jwt.WithIssuedAt(),
		jwt.WithLeeway(10*time.Second),
	)
	
	if err != nil || token == nil || !token.Valid {
		return 0, ErrInvalidToken
	}
	
	// Extract and parse the Subject claim back into a numeric user ID
	id, err := strconv.ParseUint(claims.Subject, 10, 64)
	if err != nil || id == 0 {
		return 0, ErrInvalidToken
	}
	
	return id, nil
}

// Issue creates and signs a new JWT token for a given user ID using the manager's configuration.
func (m *AccessTokenManager) Issue(userID uint64) (string, error) {
	if userID == 0 {
		return "", ErrInvalidInput
	}
	
	now := time.Now().UTC()
	
	// Define standard registered JWT claims including issuance, validity window, and expiration
	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatUint(userID, 10), 
		Issuer:    m.issuer,
		Audience:  jwt.ClaimStrings{m.audience},
		IssuedAt:  jwt.NewNumericDate(now), 
		NotBefore: jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
	}
	
	// Sign and generate the token string using HMAC-SHA256 (HS256)
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.key)
}