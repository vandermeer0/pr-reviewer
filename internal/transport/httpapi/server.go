// Package httpapi содержит HTTP-хендлеры для сервиса назначения ревьюеров на PR
package httpapi

import (
	"net/http"

	"github.com/vandermeer0/pr-reviewer/internal/usecase"
)

// Server HTTP-адаптер который предоставляет доменные сервисы как JSON-API
type Server struct {
	teamService            usecase.TeamService
	userService            usecase.UserService
	prService              usecase.PullRequestService
	statsService           usecase.StatsService
	teamMaintenanceService usecase.TeamMaintenanceService
}

// NewServer conсоздаёт HTTP-сервер с переданными доменными сервисами
func NewServer(
	teamSvc usecase.TeamService,
	userSvc usecase.UserService,
	prSvc usecase.PullRequestService,
	statsSvc usecase.StatsService,
	teamMaintSvc usecase.TeamMaintenanceService,
) *Server {
	return &Server{
		teamService:            teamSvc,
		userService:            userSvc,
		prService:              prSvc,
		statsService:           statsSvc,
		teamMaintenanceService: teamMaintSvc,
	}
}

// RegisterRoutes регистрирует все HTTP-эндпоинты на переданном ServeMux
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/healthz", s.handleHealth)

	mux.HandleFunc("/team/add", s.handleTeamAdd)
	mux.HandleFunc("/team/get", s.handleTeamGet)
	mux.HandleFunc("/team/deactivateMembers", s.handleTeamDeactivateMembers)

	mux.HandleFunc("/users/setIsActive", s.handleSetIsActive)
	mux.HandleFunc("/users/getReview", s.handleGetUserReview)

	mux.HandleFunc("/pullRequest/create", s.handlePullRequestCreate)
	mux.HandleFunc("/pullRequest/merge", s.handlePullRequestMerge)
	mux.HandleFunc("/pullRequest/reassign", s.handlePullRequestReassign)

	mux.HandleFunc("/stats/reviewers", s.handleStatsReviewers)
}
