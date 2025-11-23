package handlers

import (
	"pr-reviewer/internal/service"

	"github.com/gorilla/mux"
)

func NewRouter(service *service.Service) *mux.Router {
	handlers := NewHandlers(service)

	router := mux.NewRouter()

	router.HandleFunc("/team/add", handlers.AddTeam).Methods("POST")
	router.HandleFunc("/team/get", handlers.GetTeam).Methods("GET")

	router.HandleFunc("/users/setIsActive", handlers.SetUserActive).Methods("POST")
	router.HandleFunc("/users/getReview", handlers.GetUserReviewPRs).Methods("GET")

	router.HandleFunc("/pullRequest/create", handlers.CreatePR).Methods("POST")
	router.HandleFunc("/pullRequest/merge", handlers.MergePR).Methods("POST")
	router.HandleFunc("/pullRequest/reassign", handlers.ReassignReviewer).Methods("POST")

	router.HandleFunc("/health", handlers.HealthCheck).Methods("GET")

	return router
}
