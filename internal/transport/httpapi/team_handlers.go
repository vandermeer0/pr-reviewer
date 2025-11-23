package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/vandermeer0/pr-reviewer/internal/usecase"
)

func (s *Server) handleTeamAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	defer closeRequestBody(r)

	var dto TeamDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	if dto.TeamName == "" {
		http.Error(w, "team_name is required", http.StatusBadRequest)
		return
	}

	members := make([]usecase.CreateTeamMemberInput, 0, len(dto.Members))
	for _, m := range dto.Members {
		members = append(members, usecase.CreateTeamMemberInput{
			UserID:   m.UserID,
			Username: m.Username,
			IsActive: m.IsActive,
		})
	}

	team, err := s.teamService.CreateTeam(r.Context(), dto.TeamName, members)
	if err != nil {
		s.handleError(w, err)
		return
	}

	resp := struct {
		Team *TeamDTO `json:"team"`
	}{
		Team: teamToDTO(team),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func (s *Server) handleTeamGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		http.Error(w, "team_name query parameter is required", http.StatusBadRequest)
		return
	}

	team, err := s.teamService.GetTeam(r.Context(), teamName)
	if err != nil {
		s.handleError(w, err)
		return
	}

	dto := teamToDTO(team)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(dto); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func (s *Server) handleTeamDeactivateMembers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	defer closeRequestBody(r)

	var req teamDeactivateMembersRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.TeamName == "" {
		http.Error(w, "team_name is required", http.StatusBadRequest)
		return
	}

	res, err := s.teamMaintenanceService.DeactivateTeamMembers(r.Context(), req.TeamName)
	if err != nil {
		http.Error(w, "failed to deactivate team members", http.StatusInternalServerError)
		return
	}

	resp := teamDeactivateMembersResponse{
		TeamName:             res.TeamName,
		DeactivatedUsers:     res.DeactivatedUsers,
		RemovedAssignments:   res.RemovedAssignments,
		NewAssignments:       res.NewAssignments,
		AffectedPullRequests: res.AffectedPullRequests,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
