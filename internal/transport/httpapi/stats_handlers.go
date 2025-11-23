package httpapi

import (
	"encoding/json"
	"net/http"
)

func (s *Server) handleStatsReviewers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	stats, err := s.statsService.GetReviewerStats(r.Context())
	if err != nil {
		http.Error(w, "failed to query stats", http.StatusInternalServerError)
		return
	}

	reviewers := make([]ReviewerStatDTO, 0, len(stats))
	for _, st := range stats {
		reviewers = append(reviewers, ReviewerStatDTO{
			UserID:      st.UserID,
			Username:    st.Username,
			Assignments: st.Assignments,
		})
	}

	resp := struct {
		Reviewers []ReviewerStatDTO `json:"reviewers"`
	}{
		Reviewers: reviewers,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
