package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
)

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return false
	}
	return true
}

func closeRequestBody(r *http.Request) {
	if r.Body == nil {
		return
	}
	if err := r.Body.Close(); err != nil {
		log.Printf("failed to close request body: %v", err)
	}
}
