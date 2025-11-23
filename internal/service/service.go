package service

import (
	"errors"
	"math/rand"
	"pr-reviewer/internal/models"
	"pr-reviewer/internal/store"
	"time"
)

type Service struct {
	Store store.Store
}

func NewService(store store.Store) *Service {
	rand.Seed(time.Now().UnixNano())
	return &Service{Store: store}
}

func (s *Service) AssignReviewers(authorID string) ([]string, error) {
	author, err := s.Store.GetUser(authorID)
	if err != nil {
		return nil, err
	}

	teamMembers, err := s.Store.GetActiveTeamMembers(author.TeamName, authorID)
	if err != nil {
		return nil, err
	}

	reviewers := make([]string, 0, 2)
	shuffled := shuffleUsers(teamMembers)

	for i := 0; i < len(shuffled) && i < 2; i++ {
		reviewers = append(reviewers, shuffled[i].UserID)
	}

	return reviewers, nil
}

func (s *Service) ReassignReviewer(prID, oldUserID string) (string, error) {
	pr, err := s.Store.GetPR(prID)
	if err != nil {
		return "", err
	}

	if pr.Status == "MERGED" {
		return "", errors.New("PR_MERGED")
	}

	isAssigned, err := s.Store.IsUserAssignedToPR(prID, oldUserID)
	if err != nil {
		return "", err
	}
	if !isAssigned {
		return "", errors.New("NOT_ASSIGNED")
	}

	oldUser, err := s.Store.GetUser(oldUserID)
	if err != nil {
		return "", err
	}

	candidates, err := s.Store.GetActiveTeamMembers(oldUser.TeamName, "")
	if err != nil {
		return "", err
	}

	availableCandidates := make([]*models.User, 0)
	for _, candidate := range candidates {
		if candidate.UserID == pr.AuthorID {
			continue
		}

		isAssigned, err := s.Store.IsUserAssignedToPR(prID, candidate.UserID)
		if err != nil {
			return "", err
		}
		if !isAssigned {
			availableCandidates = append(availableCandidates, candidate)
		}
	}

	if len(availableCandidates) == 0 {
		return "", errors.New("NO_CANDIDATE")
	}

	newReviewer := availableCandidates[rand.Intn(len(availableCandidates))]

	newReviewers := make([]string, 0, len(pr.AssignedReviewers))
	for _, reviewer := range pr.AssignedReviewers {
		if reviewer == oldUserID {
			newReviewers = append(newReviewers, newReviewer.UserID)
		} else {
			newReviewers = append(newReviewers, reviewer)
		}
	}

	err = s.Store.UpdatePRReviewers(prID, newReviewers)
	if err != nil {
		return "", err
	}

	return newReviewer.UserID, nil
}

func shuffleUsers(users []*models.User) []*models.User {
	shuffled := make([]*models.User, len(users))
	copy(shuffled, users)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	return shuffled
}
