package repository

import (
	"context"
	"errors"

	"aws-local-test/internal/model"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrInvalidUser  = errors.New("invalid user data")
)

// UserRepository defines the contract for user storage operations.
type UserRepository interface {
	Create(ctx context.Context, user model.User) error
	GetByID(ctx context.Context, id string) (*model.User, error)
	UpdateUser(ctx context.Context, user model.User) error
	DeleteUser(ctx context.Context, id string) error
	ListUsers(ctx context.Context) ([]model.User, error)
}
