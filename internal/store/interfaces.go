package store

import "pr-reviewer/internal/models"

type TeamRepository interface {
	CreateTeam(team *models.Team) error
	GetTeam(teamName string) (*models.Team, error)
}

type UserRepository interface {
	UpdateUserActive(userID string, isActive bool) (*models.User, error)
	GetUser(userID string) (*models.User, error)
	GetActiveTeamMembers(teamName string, excludeUserID string) ([]*models.User, error)
}

type PRRepository interface {
	CreatePR(pr *models.PullRequest) error
	GetPR(prID string) (*models.PullRequest, error)
	MergePR(prID string) error
	UpdatePRReviewers(prID string, reviewers []string) error
	GetUserReviewPRs(userID string) ([]*models.PullRequestShort, error)
	IsUserAssignedToPR(prID, userID string) (bool, error)
}

type Store interface {
	TeamRepository
	UserRepository
	PRRepository
	Close() error
}
