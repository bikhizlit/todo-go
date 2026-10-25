package tasks

import (
	"context"
	"strings"
	"unicode/utf8"

)

type taskStore interface {
    List(ctx context.Context) ([]Task, error)
    Get(ctx context.Context, id uint64) (Task, error)
    Create(ctx context.Context, input CreateTaskInput) (Task, error)
    Update(ctx context.Context, task Task) (Task, error)
    Delete(ctx context.Context, id uint64) error
}
type TaskService struct {
	repo taskStore
}

func NewTaskService(repo taskStore) *TaskService {
	return &TaskService{repo: repo}
}

func (s *TaskService) List(ctx context.Context) ([]Task, error) {
	return s.repo.List(ctx)
}

func (s *TaskService) Get(ctx context.Context, id uint64) (Task, error) {
	return s.repo.Get(ctx, id)
}

func (s *TaskService) Create(
	ctx context.Context,
	input CreateTaskInput,
) (Task, error) {
	title, err := validateTitle(input.Title)
	if err != nil {
		return Task{}, err
	}

	input.Title = title

	return s.repo.Create(ctx, input)
}

func (s *TaskService) Update(
	ctx context.Context,
	id uint64,
	input UpdateTaskInput,
) (Task, error) {
	current, err := s.repo.Get(ctx, id)
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

	return s.repo.Update(ctx, current)
}

func (s *TaskService) Delete(ctx context.Context, id uint64) error {
	return s.repo.Delete(ctx, id)
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
