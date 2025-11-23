package store

import (
	"database/sql"
	"errors"
	"fmt"
	"pr-reviewer/internal/models"

	_ "github.com/lib/pq"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(connStr string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) CreateTeam(team *models.Team) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var exists bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM teams WHERE team_name = $1)", team.TeamName).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("TEAM_EXISTS")
	}

	_, err = tx.Exec("INSERT INTO teams (team_name) VALUES ($1)", team.TeamName)
	if err != nil {
		return err
	}

	for _, member := range team.Members {
		_, err = tx.Exec(`
			INSERT INTO users (user_id, username, team_name, is_active) 
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (user_id) DO UPDATE SET 
				username = EXCLUDED.username,
				team_name = EXCLUDED.team_name,
				is_active = EXCLUDED.is_active,
				updated_at = NOW()
		`, member.UserID, member.Username, team.TeamName, member.IsActive)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *PostgresStore) GetTeam(teamName string) (*models.Team, error) {
	var team models.Team
	team.TeamName = teamName

	rows, err := s.db.Query(`
		SELECT user_id, username, is_active 
		FROM users 
		WHERE team_name = $1
		ORDER BY user_id
	`, teamName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var member models.TeamMember
		if err := rows.Scan(&member.UserID, &member.Username, &member.IsActive); err != nil {
			return nil, err
		}
		team.Members = append(team.Members, member)
	}

	if len(team.Members) == 0 {
		return nil, errors.New("NOT_FOUND")
	}

	return &team, nil
}

func (s *PostgresStore) UpdateUserActive(userID string, isActive bool) (*models.User, error) {
	var user models.User
	err := s.db.QueryRow(`
		UPDATE users 
		SET is_active = $1, updated_at = NOW() 
		WHERE user_id = $2
		RETURNING user_id, username, team_name, is_active
	`, isActive, userID).Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("NOT_FOUND")
		}
		return nil, err
	}

	return &user, nil
}

func (s *PostgresStore) GetUser(userID string) (*models.User, error) {
	var user models.User
	err := s.db.QueryRow(`
		SELECT user_id, username, team_name, is_active 
		FROM users 
		WHERE user_id = $1
	`, userID).Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("NOT_FOUND")
		}
		return nil, err
	}

	return &user, nil
}

func (s *PostgresStore) GetActiveTeamMembers(teamName string, excludeUserID string) ([]*models.User, error) {
	rows, err := s.db.Query(`
		SELECT user_id, username, team_name, is_active 
		FROM users 
		WHERE team_name = $1 AND is_active = true AND user_id != $2
		ORDER BY user_id
	`, teamName, excludeUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive); err != nil {
			return nil, err
		}
		users = append(users, &user)
	}

	return users, nil
}

func (s *PostgresStore) CreatePR(pr *models.PullRequest) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var exists bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM pull_requests WHERE pull_request_id = $1)", pr.PullRequestID).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("PR_EXISTS")
	}

	var authorTeam string
	err = tx.QueryRow("SELECT team_name FROM users WHERE user_id = $1", pr.AuthorID).Scan(&authorTeam)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("NOT_FOUND")
		}
		return err
	}

	_, err = tx.Exec(`
		INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status)
		VALUES ($1, $2, $3, 'OPEN')
	`, pr.PullRequestID, pr.PullRequestName, pr.AuthorID)
	if err != nil {
		return err
	}

	for _, reviewerID := range pr.AssignedReviewers {
		_, err = tx.Exec(`
			INSERT INTO pull_request_reviewers (pull_request_id, user_id)
			VALUES ($1, $2)
		`, pr.PullRequestID, reviewerID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *PostgresStore) GetPR(prID string) (*models.PullRequest, error) {
	var pr models.PullRequest
	var createdAt, mergedAt sql.NullTime

	err := s.db.QueryRow(`
		SELECT pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status, pr.created_at, pr.merged_at
		FROM pull_requests pr
		WHERE pr.pull_request_id = $1
	`, prID).Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status, &createdAt, &mergedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("NOT_FOUND")
		}
		return nil, err
	}

	if createdAt.Valid {
		pr.CreatedAt = &createdAt.Time
	}
	if mergedAt.Valid {
		pr.MergedAt = &mergedAt.Time
	}

	rows, err := s.db.Query(`
		SELECT user_id 
		FROM pull_request_reviewers 
		WHERE pull_request_id = $1
		ORDER BY assigned_at
	`, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var reviewerID string
		if err := rows.Scan(&reviewerID); err != nil {
			return nil, err
		}
		pr.AssignedReviewers = append(pr.AssignedReviewers, reviewerID)
	}

	return &pr, nil
}

func (s *PostgresStore) MergePR(prID string) error {
	_, err := s.db.Exec(`
		UPDATE pull_requests 
		SET status = 'MERGED', merged_at = NOW() 
		WHERE pull_request_id = $1 AND status != 'MERGED'
	`, prID)
	return err
}

func (s *PostgresStore) UpdatePRReviewers(prID string, reviewers []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM pull_request_reviewers WHERE pull_request_id = $1", prID)
	if err != nil {
		return err
	}

	for _, reviewerID := range reviewers {
		_, err = tx.Exec(`
			INSERT INTO pull_request_reviewers (pull_request_id, user_id)
			VALUES ($1, $2)
		`, prID, reviewerID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *PostgresStore) GetUserReviewPRs(userID string) ([]*models.PullRequestShort, error) {
	rows, err := s.db.Query(`
		SELECT pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status
		FROM pull_requests pr
		JOIN pull_request_reviewers prr ON pr.pull_request_id = prr.pull_request_id
		WHERE prr.user_id = $1
		ORDER BY pr.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prs []*models.PullRequestShort
	for rows.Next() {
		var pr models.PullRequestShort
		if err := rows.Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status); err != nil {
			return nil, err
		}
		prs = append(prs, &pr)
	}

	return prs, nil
}

func (s *PostgresStore) IsUserAssignedToPR(prID, userID string) (bool, error) {
	var exists bool
	err := s.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM pull_request_reviewers 
			WHERE pull_request_id = $1 AND user_id = $2
		)
	`, prID, userID).Scan(&exists)
	return exists, err
}

func (s *PostgresStore) Close() error {
	return s.db.Close()
}
