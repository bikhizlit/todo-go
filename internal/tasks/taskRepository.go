
package tasks

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

)



type TaskRepository struct {
	db *sql.DB
}

func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) List(ctx context.Context) ([]Task, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, description, completed, created_at, updated_at
		FROM tasks
		ORDER BY created_at DESC, id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query tasks: %w", err)
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
			return nil, fmt.Errorf("scan task: %w", err)
		}

		result = append(result, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}

	return result, nil
}

func (r *TaskRepository) Get(
	ctx context.Context,
	id uint64,
) (Task, error) {
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
		return Task{}, fmt.Errorf("get task: %w", err)
	}

	return task, nil
}

func (r *TaskRepository) Create(
	ctx context.Context,
	input CreateTaskInput,
) (Task, error) {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO tasks (title, description)
		VALUES (?, ?)
	`, input.Title, input.Description)
	if err != nil {
		return Task{}, fmt.Errorf("insert task: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return Task{}, fmt.Errorf("get inserted task ID: %w", err)
	}

	return r.Get(ctx, uint64(id))
}

func (r *TaskRepository) Update(
	ctx context.Context,
	task Task,
) (Task, error) {
	_, err := r.db.ExecContext(ctx, `
		UPDATE tasks
		SET title = ?, description = ?, completed = ?
		WHERE id = ?
	`, task.Title, task.Description, task.Completed, task.ID)
	if err != nil {
		return Task{}, fmt.Errorf("update task: %w", err)
	}

	return r.Get(ctx, task.ID)
}

func (r *TaskRepository) Delete(
	ctx context.Context,
	id uint64,
) error {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM tasks
		WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check deleted task: %w", err)
	}

	if affected == 0 {
		return ErrNotFound
	}

	return nil
}
