package tasks

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"unicode/utf8"
)

var (
	ErrNotFound     = errors.New("task not found")
	ErrInvalidID    = errors.New("invalid task ID")
	ErrInvalidTitle = errors.New("title must contain 1 to 200 characters")
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func validateTitle(title string) (string, error) {
	title = strings.TrimSpace(title)

	if !utf8.ValidString(title) {
		return "", ErrInvalidTitle
	}

	length := len([]rune(title))
	if length < 1 || length > 200 {
		return "", ErrInvalidTitle
	}

	return title, nil
}

func (r *Repository) List(ctx context.Context) ([]Task, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, description, completed, created_at, updated_at
		FROM tasks
		ORDER BY created_at DESC, id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]Task, 0)

	for rows.Next() {
		var task Task

		if err := rows.Scan(
			&task.ID,
			&task.Title,
			&task.Description,
			&task.Completed,
			&task.CreatedAt,
			&task.UpdatedAt,
		); err != nil {
			return nil, err
		}

		result = append(result, task)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *Repository) Get(ctx context.Context, id uint64) (Task, error) {
	var task Task

	err := r.db.QueryRowContext(ctx, `
		SELECT id, title, description, completed, created_at, updated_at
		FROM tasks
		WHERE id = ?
	`, id).Scan(
		&task.ID,
		&task.Title,
		&task.Description,
		&task.Completed,
		&task.CreatedAt,
		&task.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return Task{}, ErrNotFound
	}
	if err != nil {
		return Task{}, err
	}

	return task, nil
}

func (r *Repository) Create(ctx context.Context, input CreateTaskInput) (Task, error) {
	title, err := validateTitle(input.Title)
	if err != nil {
		return Task{}, err
	}

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO tasks (title, description)
		VALUES (?, ?)
	`, title, input.Description)
	if err != nil {
		return Task{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return Task{}, err
	}

	return r.Get(ctx, uint64(id))
}

func (r *Repository) Update(
	ctx context.Context,
	id uint64,
	input UpdateTaskInput,
) (Task, error) {
	current, err := r.Get(ctx, id)
	if err != nil {
		return Task{}, err
	}

	if input.Title != nil {
		current.Title = *input.Title
	}

	if input.Description != nil {
		current.Description = *input.Description
	}

	if input.Completed != nil {
		current.Completed = *input.Completed
	}

	title, err := validateTitle(current.Title)
	if err != nil {
		return Task{}, err
	}
	current.Title = title

	_, err = r.db.ExecContext(ctx, `
		UPDATE tasks
		SET title = ?, description = ?, completed = ?
		WHERE id = ?
	`, current.Title, current.Description, current.Completed, id)
	if err != nil {
		return Task{}, err
	}
	return r.Get(ctx, id)
}

func (r *Repository) Delete(ctx context.Context, id uint64) error {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM tasks
		WHERE id = ?
	`, id)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func ValidateID(id int64) (uint64, error) {
	if id <= 0 {
		return 0, ErrInvalidID
	}
	return uint64(id), nil
}
