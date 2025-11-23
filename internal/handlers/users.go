package handlers

import (
	"encoding/json"
	"net/http"
)

func (h *Handlers) SetUserActive(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string `json:"user_id"`
		IsActive bool   `json:"is_active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.service.Store.UpdateUserActive(req.UserID, req.IsActive)
	if err != nil {
		if err.Error() == "NOT_FOUND" {
			sendErrorResponse(w, "NOT_FOUND", "resource not found", http.StatusNotFound)
		} else {
			sendError(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user": user,
	})
}

func (h *Handlers) GetUserReviewPRs(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		sendError(w, "user_id is required", http.StatusBadRequest)
		return
	}

	_, err := h.service.Store.GetUser(userID)
	if err != nil {
		if err.Error() == "NOT_FOUND" {
			sendErrorResponse(w, "NOT_FOUND", "resource not found", http.StatusNotFound)
		} else {
			sendError(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	prs, err := h.service.Store.GetUserReviewPRs(userID)
	if err != nil {
		sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id":       userID,
		"pull_requests": prs,
	})
}
