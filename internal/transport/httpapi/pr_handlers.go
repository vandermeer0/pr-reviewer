package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/vandermeer0/pr-reviewer/internal/usecase"
)

func (s *Server) handlePullRequestCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	defer closeRequestBody(r)

	var req pullRequestCreateRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.PullRequestID == "" || req.PullRequestName == "" || req.AuthorID == "" {
		http.Error(w, "pull_request_id, pull_request_name and author_id are required", http.StatusBadRequest)
		return
	}

	input := usecase.PullRequestCreateInput{
		ID:       req.PullRequestID,
		Name:     req.PullRequestName,
		AuthorID: req.AuthorID,
	}

	pr, err := s.prService.Create(r.Context(), input)
	if err != nil {
		s.handleError(w, err)
		return
	}

	resp := struct {
		PR *PullRequestDTO `json:"pr"`
	}{
		PR: pullRequestToDTO(pr),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func (s *Server) handlePullRequestMerge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	defer closeRequestBody(r)

	var req pullRequestMergeRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.PullRequestID == "" {
		http.Error(w, "pull_request_id is required", http.StatusBadRequest)
		return
	}

	pr, err := s.prService.Merge(r.Context(), req.PullRequestID)
	if err != nil {
		s.handleError(w, err)
		return
	}

	resp := struct {
		PR *PullRequestDTO `json:"pr"`
	}{
		PR: pullRequestToDTO(pr),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func (s *Server) handlePullRequestReassign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	defer closeRequestBody(r)

	var req pullRequestReassignRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.PullRequestID == "" || req.OldUserID == "" {
		http.Error(w, "pull_request_id and old_user_id are required", http.StatusBadRequest)
		return
	}

	pr, newReviewerID, err := s.prService.ReassignReviewer(r.Context(), req.PullRequestID, req.OldUserID)
	if err != nil {
		s.handleError(w, err)
		return
	}

	resp := struct {
		PR         *PullRequestDTO `json:"pr"`
		ReplacedBy string          `json:"replaced_by"`
	}{
		PR:         pullRequestToDTO(pr),
		ReplacedBy: newReviewerID,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
