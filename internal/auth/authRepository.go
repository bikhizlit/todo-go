package auth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)



type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateUser(
	ctx context.Context, 
 name, email, passwordHash string) (*User, error) {
		result, err := r.db.ExecContext(ctx, `
		INSERT INTO users (name, email, passwordHash)
		VALUES (?, ?, ?)
		`,name, email, passwordHash)
		if err != nil {
			return nil, fmt.Errorf("Insert user: %w", err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("get inserted user ID: %w", err)
		}

		return r.FindById(ctx, uint64(id))
	}

func (r *UserRepository) FindById(
	ctx context.Context,
	id uint64,
) (*User, error) {
	return scanUser(r.db.QueryRowContext(ctx, `
	SELECT id, name, email, password_hash, created_at, updated_at
	FROM users
	WHERE id = ?
	LIMIT 1
	`, id))
}

func (r *UserRepository) FindByEmail(
	ctx context.Context,
	email string,
) (*User, error) {
	return scanUser(r.db.QueryRowContext(ctx, `
	SELECT id, name, email, password_hash, created_at, updated_at
	FROM users
	WHERE email = ?
	LIMIT 1
	`, email))
}

func (r *UserRepository) FindRefreshUserID(
	ctx context.Context,
	rawToken string,
) (uint64, error) {
	var userID uint64

	err := r.db.QueryRowContext(ctx, `
		SELECT user_id
		FROM refresh_sessions
		WHERE token_hash = ?
		  AND revoked_at IS NULL
		  AND expires_at > UTC_TIMESTAMP(6)
		LIMIT 1
	`, hashRefreshToken(rawToken)).Scan(&userID)

	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrInvalidRefresh
	}
	if err != nil {
		return 0, fmt.Errorf("find refresh token owner: %w", err)
	}

	return userID, nil
}


// helper functions: scanUser scans while hashRefreshToken computes hex tokens
func scanUser(row *sql.Row) (*User, error) {
	var user User
	err := row.Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("Scan user: %w", err)
	}
	return &user, nil
}

func hashRefreshToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}


func (r *UserRepository) CreateRefreshSession(
	ctx context.Context,
	userID uint64,
	rawToken string,
	expiresAt time.Time,
) error {
	_, err := r.db.ExecContext(ctx, `
	INSERT INTO refresh_sessions (user_id, token_hash, expires_at)
	VAlUES (?, ?, ?)
	`, userID, hashRefreshToken(rawToken), expiresAt.UTC())

	if err != nil {
		return fmt.Errorf("create refresh session: %w", err)
	}

	return nil
}


func (r *UserRepository) RotateRefreshSession(
	ctx context.Context,
	oldRawToken, newRawToken string,
	newExpiresAt time.Time,
) (uint64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin refresh transaction: %w", err)
	}
	defer tx.Rollback()

	var oldID, userID uint64

	err = tx.QueryRowContext(ctx, `
	SELECT id, user_id
	FROM refresh_sessions
	WHERE token_hash = ?
	AND revoked_at IS NULL
	AND expires_at > UTC_TIMESTAMP(6)
	FOR UPDATE
	`, hashRefreshToken(oldRawToken)).Scan(&oldID, &userID)

	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrInvalidRefresh
	}

	if err != nil {
		return 0, fmt.Errorf("find refresh session: %w", err)
	}

	result, err := tx.ExecContext(ctx, `
		INSERT INTO refresh_sessions (user_id, token_hash, expires_at)
		VALUES (?, ?, ?)
	`, userID, hashRefreshToken(newRawToken), newExpiresAt.UTC())

	if err != nil {
		return 0, fmt.Errorf("Create rotated session: %w", err)
	}

	newID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get replacement session ID: %w", err)
	}

	result, err = tx.ExecContext(ctx, `
	UPDATE refresh_sessions
	SET revoked_at = UTC_TIMESTAMP(6),
	replaced_by_session_id = ?
	WHERE id = ?
	AND revoked_at IS NULL
	`, newID, oldID)

	if err != nil {
		return 0, fmt.Errorf("revoke old session: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("check revoked session: %w", err)
	}

	if affected != 1 {
		return 0, ErrInvalidRefresh
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit refresh transaction: %w", err)
	}

	return userID, nil
}


func (r *UserRepository) RevokeRefreshSession(
	ctx context.Context,
	rawToken string,
) error {
	_, err := r.db.ExecContext(ctx, `
	UPDATE refresh_sessions
	SET revoked_at = UTC_TIMESTAMP(6)
	WHERE token_hash = ?
	AND revoked_at IS NULL
	`, hashRefreshToken(rawToken))

	if err != nil {
		return fmt.Errorf("revoke refresh session: %w", err)
	}

	return nil
}