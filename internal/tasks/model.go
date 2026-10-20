
package tasks

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrNotFound     = errors.New("task not found")
	ErrInvalidID    = errors.New("invalid task ID")
	ErrInvalidTitle = errors.New("title must contain 1 to 200 characters")
)

type Task struct {
	ID          uint64    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateTaskInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type UpdateTaskInput struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Completed   *bool   `json:"completed"`
}

func ValidateID(id int64) (uint64, error) {
	if id <= 0 {
		return 0, fmt.Errorf("%w: must be positive", ErrInvalidID)
	}

	return uint64(id), nil
}
