package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/vandermeer0/pr-reviewer/internal/usecase"
)

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorResponse struct {
	Error errorBody `json:"error"`
}

func (s *Server) handleError(w http.ResponseWriter, err error) {
	var de *usecase.DomainError
	if errors.As(err, &de) {
		status := httpStatusForCode(de.Code)
		writeDomainError(w, status, de)
		return
	}

	http.Error(w, "internal server error", http.StatusInternalServerError)
}

func httpStatusForCode(code usecase.ErrorCode) int {
	switch code {
	case usecase.ErrorCodeTeamExists:
		return http.StatusBadRequest
	case usecase.ErrorCodeNotFound:
		return http.StatusNotFound
	case usecase.ErrorCodePRExists,
		usecase.ErrorCodePRMerged,
		usecase.ErrorCodeNotAssigned,
		usecase.ErrorCodeNoCandidate:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func writeDomainError(w http.ResponseWriter, status int, de *usecase.DomainError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resp := errorResponse{
		Error: errorBody{
			Code:    string(de.Code),
			Message: de.Message,
		},
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "failed to encode error response", http.StatusInternalServerError)
	}
}
