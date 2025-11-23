package usecase

import (
	"context"

	"github.com/vandermeer0/pr-reviewer/internal/entity"
)

// CreateTeamMemberInput - данные для создания команды
type CreateTeamMemberInput struct {
	UserID   string
	Username string
	IsActive bool
}

// PullRequestCreateInput - данные для создания PR
type PullRequestCreateInput struct {
	ID       string
	Name     string
	AuthorID string
}

// TeamService описывает операции с командами
type TeamService interface {
	// CreateTeam создаёт команду и юзеров
	CreateTeam(ctx context.Context, teamName string, members []CreateTeamMemberInput) (*entity.Team, error)

	// GetTeam возвращает команду по имени или NOT_FOUND
	GetTeam(ctx context.Context, teamName string) (*entity.Team, error)
}

// UserService описывает операции с юзерами
type UserService interface {
	// SetIsActive меняет флаг активности юзера
	SetIsActive(ctx context.Context, userID string, isActive bool) (*entity.User, error)
}

// PullRequestService описывает операции с PR
type PullRequestService interface {
	// Create создаёт PR и назначает до двух ревьюверов
	Create(ctx context.Context, input PullRequestCreateInput) (*entity.PullRequest, error)

	// Merge идемпотентно помечает PR как MERGED
	Merge(ctx context.Context, prID string) (*entity.PullRequest, error)

	// ReassignReviewer заменяет ревьювера на другого из его команды
	ReassignReviewer(ctx context.Context, prID string, oldReviewerID string) (*entity.PullRequest, string, error)

	// GetByReviewer возвращает PRы где пользователь ревьювер
	GetByReviewer(ctx context.Context, reviewerID string) ([]*entity.PullRequest, error)
}
