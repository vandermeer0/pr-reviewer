// Package repo описывает контракты хранилищ
package repo

import (
	"context"
	"errors"

	"github.com/vandermeer0/pr-reviewer/internal/entity"
)

// ErrNotFound означает что сущность не найдена
var ErrNotFound = errors.New("not found")

// ErrAlreadyExists означает что сущность уже существует
var ErrAlreadyExists = errors.New("already exists")

// UserRepository описывает работу с пользователями
type UserRepository interface {
	Save(ctx context.Context, user *entity.User) error
	GetByID(ctx context.Context, id string) (*entity.User, error)
	SaveBatch(ctx context.Context, users []*entity.User) error
}

// TeamRepository описывает работу с командами
type TeamRepository interface {
	Save(ctx context.Context, team *entity.Team) error
	GetByName(ctx context.Context, name string) (*entity.Team, error)
}

// PullRequestRepository описывает работу с PR
type PullRequestRepository interface {
	Save(ctx context.Context, pr *entity.PullRequest) error
	GetByID(ctx context.Context, id string) (*entity.PullRequest, error)
	Update(ctx context.Context, pr *entity.PullRequest) error
	GetByReviewerID(ctx context.Context, reviewerID string) ([]*entity.PullRequest, error)
}
