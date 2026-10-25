package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

// Constants defining the cryptographic parameters for Argon2id hashing
const (
	saltSize           = 16
	hashSize    uint32 = 32
	memoryKiB   uint32 = 32 * 1024
	iterations  uint32 = 2
	parallelism uint8  = 1
)

// validateNewPassword ensures the password meets security requirements:
// valid UTF-8, at least 15 characters, and no longer than 256 bytes.
func validateNewPassword(password string) bool {
	return utf8.ValidString(password) && utf8.RuneCountInString(password) >= 8 && len(password) <= 256
}

// HashPassword takes a plaintext password, validates it, generates a secure random salt,
// and hashes it using the Argon2id key derivation function.
func HashPassword(password string) (string, error) {

	// Check if the password meets the validation rules
	if !validateNewPassword(password) {
		return "", ErrInvalidInput
	}

	// Generate a cryptographically secure random salt
	salt := make([]byte, saltSize)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}

	// Derive the key using Argon2id with our defined parameters
	hash := argon2.IDKey([]byte(password), salt, iterations, memoryKiB, parallelism, hashSize)

	// Format the resulting hash string to include parameters, salt, and the hash itself
	return fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		memoryKiB,
		iterations,
		parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

// VerifyPassword checks if a plaintext password matches a stored encoded Argon2id hash.
func VerifyPassword(password, encoded string) (bool, error) {

	// Split the encoded string to extract algorithm parts, parameters, salt, and hash
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return false, errors.New("unsupported stored password hash")
	}

	// Extract and validate the parameters component (memory, iterations, parallelism)
	params := strings.Split(parts[3], ",")
	if len(params) != 3 {
		return false, errors.New("bad password hash parameter")
	}

	// Initialize the 'parsed' map to store the key-value pairs of the hash parameters
	// extracted from the encoded string (e.g., 'm' for memory, 't' for iterations, 'p' for parallelism)
	parsed := map[string]uint64{}
	for _, part := range params {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			return false, errors.New("bad password hash parameter")
		}

		// Check for any duplicate configuration keys in the parameter string
		if _, exits := parsed[kv[0]]; exits {
			return false, errors.New("duplicate parameter")
		}

		// Parse the parameter value [string to uint64] and store it in the map
		var val uint64
		_, err := fmt.Sscanf(kv[1], "%d", &val)
		if err != nil {
			return false, errors.New("bad password hash parameter value")
		}
		parsed[kv[0]] = val

	}

	// Decode the base64-encoded salt
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) != saltSize {
		return false, errors.New("bad password salt")
	}

	// Decode the base64-encoded expected hash
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(expected) != int(hashSize) {
		return false, errors.New("bad password hash")
	}

	// Re-hash the provided plaintext password using the extracted parameters and salt
	actual := argon2.IDKey([]byte(password), salt, uint32(parsed["t"]), uint32(parsed["m"]), uint8(parsed["p"]), uint32(len(expected)))

	// Use constant-time comparison to prevent timing attacks when comparing hashes
	return subtle.ConstantTimeCompare(actual, expected) == 1, nil

}
