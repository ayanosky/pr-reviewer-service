package handlers

import (
	"encoding/json"
	"net/http"
	"pr-reviewer/internal/models"
)

func (h *Handlers) AddTeam(w http.ResponseWriter, r *http.Request) {
	var team models.Team
	if err := json.NewDecoder(r.Body).Decode(&team); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.service.Store.CreateTeam(&team); err != nil {
		switch err.Error() {
		case "TEAM_EXISTS":
			sendErrorResponse(w, "TEAM_EXISTS", "team_name already exists", http.StatusBadRequest)
		default:
			sendError(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"team": team,
	})
}

func (h *Handlers) GetTeam(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		sendError(w, "team_name is required", http.StatusBadRequest)
		return
	}

	team, err := h.service.Store.GetTeam(teamName)
	if err != nil {
		if err.Error() == "NOT_FOUND" {
			sendErrorResponse(w, "NOT_FOUND", "resource not found", http.StatusNotFound)
		} else {
			sendError(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(team)
}
