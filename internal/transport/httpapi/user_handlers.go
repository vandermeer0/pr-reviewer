package httpapi

import (
	"encoding/json"
	"net/http"
)

func (s *Server) handleSetIsActive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	defer closeRequestBody(r)

	var req setIsActiveRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.UserID == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	user, err := s.userService.SetIsActive(r.Context(), req.UserID, req.IsActive)
	if err != nil {
		s.handleError(w, err)
		return
	}

	resp := struct {
		User *UserDTO `json:"user"`
	}{
		User: userToDTO(user),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func (s *Server) handleGetUserReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id query parameter is required", http.StatusBadRequest)
		return
	}

	prs, err := s.prService.GetByReviewer(r.Context(), userID)
	if err != nil {
		s.handleError(w, err)
		return
	}

	short := make([]PullRequestShortDTO, 0, len(prs))
	for _, pr := range prs {
		if pr == nil {
			continue
		}
		short = append(short, PullRequestShortDTO{
			PullRequestID:   pr.ID,
			PullRequestName: pr.Name,
			AuthorID:        pr.AuthorID,
			Status:          string(pr.Status),
		})
	}

	resp := struct {
		UserID       string                `json:"user_id"`
		PullRequests []PullRequestShortDTO `json:"pull_requests"`
	}{
		UserID:       userID,
		PullRequests: short,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
