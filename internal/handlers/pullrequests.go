package handlers

import (
	"encoding/json"
	"net/http"
	"pr-reviewer/internal/models"
)

func (h *Handlers) CreatePR(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PullRequestID   string `json:"pull_request_id"`
		PullRequestName string `json:"pull_request_name"`
		AuthorID        string `json:"author_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	reviewers, err := h.service.AssignReviewers(req.AuthorID)
	if err != nil {
		if err.Error() == "NOT_FOUND" {
			sendErrorResponse(w, "NOT_FOUND", "resource not found", http.StatusNotFound)
		} else {
			sendError(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	pr := &models.PullRequest{
		PullRequestID:     req.PullRequestID,
		PullRequestName:   req.PullRequestName,
		AuthorID:          req.AuthorID,
		Status:            "OPEN",
		AssignedReviewers: reviewers,
	}

	if err := h.service.Store.CreatePR(pr); err != nil {
		switch err.Error() {
		case "PR_EXISTS":
			sendErrorResponse(w, "PR_EXISTS", "PR id already exists", http.StatusConflict)
		case "NOT_FOUND":
			sendErrorResponse(w, "NOT_FOUND", "resource not found", http.StatusNotFound)
		default:
			sendError(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pr": pr,
	})
}

func (h *Handlers) MergePR(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PullRequestID string `json:"pull_request_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	pr, err := h.service.Store.GetPR(req.PullRequestID)
	if err != nil {
		if err.Error() == "NOT_FOUND" {
			sendErrorResponse(w, "NOT_FOUND", "resource not found", http.StatusNotFound)
		} else {
			sendError(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if pr.Status != "MERGED" {
		if err := h.service.Store.MergePR(req.PullRequestID); err != nil {
			sendError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		pr, _ = h.service.Store.GetPR(req.PullRequestID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pr": pr,
	})
}

func (h *Handlers) ReassignReviewer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PullRequestID string `json:"pull_request_id"`
		OldUserID     string `json:"old_user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	newUserID, err := h.service.ReassignReviewer(req.PullRequestID, req.OldUserID)
	if err != nil {
		switch err.Error() {
		case "NOT_FOUND":
			sendErrorResponse(w, "NOT_FOUND", "resource not found", http.StatusNotFound)
		case "PR_MERGED":
			sendErrorResponse(w, "PR_MERGED", "cannot reassign on merged PR", http.StatusConflict)
		case "NOT_ASSIGNED":
			sendErrorResponse(w, "NOT_ASSIGNED", "reviewer is not assigned to this PR", http.StatusConflict)
		case "NO_CANDIDATE":
			sendErrorResponse(w, "NO_CANDIDATE", "no active replacement candidate in team", http.StatusConflict)
		default:
			sendError(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	pr, _ := h.service.Store.GetPR(req.PullRequestID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pr":          pr,
		"replaced_by": newUserID,
	})
}
