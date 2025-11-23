package httpapi

import (
	"time"

	"github.com/vandermeer0/pr-reviewer/internal/entity"
)

// TeamMemberDTO представляет участника команды в HTTP JSON
type TeamMemberDTO struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

// TeamDTO представляет команду и её участников в HTTP JSON
type TeamDTO struct {
	TeamName string          `json:"team_name"`
	Members  []TeamMemberDTO `json:"members"`
}

// UserDTO представляет пользователя в HTTP JSON
type UserDTO struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	TeamName string `json:"team_name"`
	IsActive bool   `json:"is_active"`
}

// PullRequestDTO представляет PR со списком ревьюверов в HTTP JSON
type PullRequestDTO struct {
	PullRequestID     string     `json:"pull_request_id"`
	PullRequestName   string     `json:"pull_request_name"`
	AuthorID          string     `json:"author_id"`
	Status            string     `json:"status"`
	AssignedReviewers []string   `json:"assigned_reviewers"`
	CreatedAt         time.Time  `json:"createdAt"`
	MergedAt          *time.Time `json:"mergedAt,omitempty"`
}

// PullRequestShortDTO представляет краткие данные о PR для ревьювера
type PullRequestShortDTO struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
	Status          string `json:"status"`
}

// ReviewerStatDTO представляет статистику ревьювера в HTTP JSON
type ReviewerStatDTO struct {
	UserID      string `json:"user_id"`
	Username    string `json:"username"`
	Assignments int64  `json:"assignments"`
}

// teamDeactivateMembersRequest описывает запрос на массовую деактивацию
type teamDeactivateMembersRequest struct {
	TeamName string `json:"team_name"`
}

// teamDeactivateMembersResponse описывает результат массовой деактивации
type teamDeactivateMembersResponse struct {
	TeamName             string `json:"team_name"`
	DeactivatedUsers     int64  `json:"deactivated_users"`
	RemovedAssignments   int64  `json:"removed_assignments"`
	NewAssignments       int64  `json:"new_assignments"`
	AffectedPullRequests int    `json:"affected_pull_requests"`
}

type setIsActiveRequest struct {
	UserID   string `json:"user_id"`
	IsActive bool   `json:"is_active"`
}

type pullRequestCreateRequest struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
}

type pullRequestMergeRequest struct {
	PullRequestID string `json:"pull_request_id"`
}

type pullRequestReassignRequest struct {
	PullRequestID string `json:"pull_request_id"`
	OldUserID     string `json:"old_user_id"`
}

func teamToDTO(team *entity.Team) *TeamDTO {
	if team == nil {
		return nil
	}
	members := make([]TeamMemberDTO, 0, len(team.Members))
	for _, m := range team.Members {
		if m == nil {
			continue
		}
		members = append(members, TeamMemberDTO{
			UserID:   m.ID,
			Username: m.Username,
			IsActive: m.IsActive,
		})
	}
	return &TeamDTO{
		TeamName: team.Name,
		Members:  members,
	}
}

func userToDTO(u *entity.User) *UserDTO {
	if u == nil {
		return nil
	}
	return &UserDTO{
		UserID:   u.ID,
		Username: u.Username,
		TeamName: u.TeamName,
		IsActive: u.IsActive,
	}
}

func pullRequestToDTO(pr *entity.PullRequest) *PullRequestDTO {
	if pr == nil {
		return nil
	}

	reviewers := make([]string, 0, len(pr.Reviewers))
	reviewers = append(reviewers, pr.Reviewers...)

	return &PullRequestDTO{
		PullRequestID:     pr.ID,
		PullRequestName:   pr.Name,
		AuthorID:          pr.AuthorID,
		Status:            string(pr.Status),
		AssignedReviewers: reviewers,
		CreatedAt:         pr.CreatedAt,
		MergedAt:          pr.MergedAt,
	}
}
