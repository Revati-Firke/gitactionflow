package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository is the locally connected GitHub repository.
type Repository struct {
	ID                 uuid.UUID
	UserID             uuid.UUID
	GitHubRepositoryID int64
	Name               string
	FullName           string
	OwnerLogin         string
	DefaultBranch      string
	HTMLURL            string
	Private            bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// Repositories persists connected repositories.
type Repositories struct {
	pool *pgxpool.Pool
}

func NewRepositories(pool *pgxpool.Pool) *Repositories {
	return &Repositories{pool: pool}
}

// GetByUserID returns the connected repository for a user, if any.
func (s *Repositories) GetByUserID(ctx context.Context, userID uuid.UUID) (Repository, error) {
	const q = `
SELECT id, user_id, github_repository_id, name, full_name, owner_login,
       default_branch, html_url, private, created_at, updated_at
FROM repositories WHERE user_id = $1
`
	var r Repository
	err := s.pool.QueryRow(ctx, q, userID).Scan(
		&r.ID, &r.UserID, &r.GitHubRepositoryID, &r.Name, &r.FullName, &r.OwnerLogin,
		&r.DefaultBranch, &r.HTMLURL, &r.Private, &r.CreatedAt, &r.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Repository{}, ErrNotFound
	}
	if err != nil {
		return Repository{}, fmt.Errorf("get repository: %w", err)
	}
	return r, nil
}

// GetByID loads a connected repository by primary key.
func (s *Repositories) GetByID(ctx context.Context, id uuid.UUID) (Repository, error) {
	const q = `
SELECT id, user_id, github_repository_id, name, full_name, owner_login,
       default_branch, html_url, private, created_at, updated_at
FROM repositories WHERE id = $1
`
	var r Repository
	err := s.pool.QueryRow(ctx, q, id).Scan(
		&r.ID, &r.UserID, &r.GitHubRepositoryID, &r.Name, &r.FullName, &r.OwnerLogin,
		&r.DefaultBranch, &r.HTMLURL, &r.Private, &r.CreatedAt, &r.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Repository{}, ErrNotFound
	}
	if err != nil {
		return Repository{}, fmt.Errorf("get repository by id: %w", err)
	}
	return r, nil
}

// GetByGitHubID finds a connected repository by GitHub's numeric repository id.
func (s *Repositories) GetByGitHubID(ctx context.Context, githubRepoID int64) (Repository, error) {
	const q = `
SELECT id, user_id, github_repository_id, name, full_name, owner_login,
       default_branch, html_url, private, created_at, updated_at
FROM repositories WHERE github_repository_id = $1
LIMIT 1
`
	var r Repository
	err := s.pool.QueryRow(ctx, q, githubRepoID).Scan(
		&r.ID, &r.UserID, &r.GitHubRepositoryID, &r.Name, &r.FullName, &r.OwnerLogin,
		&r.DefaultBranch, &r.HTMLURL, &r.Private, &r.CreatedAt, &r.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Repository{}, ErrNotFound
	}
	if err != nil {
		return Repository{}, fmt.Errorf("get repository by github id: %w", err)
	}
	return r, nil
}

// Insert creates a connected repository. UNIQUE(user_id) enforces one-per-user.
func (s *Repositories) Insert(ctx context.Context, userID uuid.UUID, githubRepoID int64, name, fullName, ownerLogin, defaultBranch, htmlURL string, private bool) (Repository, error) {
	const q = `
INSERT INTO repositories (
    user_id, github_repository_id, name, full_name, owner_login, default_branch, html_url, private
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, user_id, github_repository_id, name, full_name, owner_login,
          default_branch, html_url, private, created_at, updated_at
`
	var r Repository
	err := s.pool.QueryRow(ctx, q, userID, githubRepoID, name, fullName, ownerLogin, defaultBranch, htmlURL, private).Scan(
		&r.ID, &r.UserID, &r.GitHubRepositoryID, &r.Name, &r.FullName, &r.OwnerLogin,
		&r.DefaultBranch, &r.HTMLURL, &r.Private, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Repository{}, ErrConflict
		}
		return Repository{}, fmt.Errorf("insert repository: %w", err)
	}
	return r, nil
}

// DeleteByUserID removes the connected repository for the user (idempotent).
func (s *Repositories) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	const q = `DELETE FROM repositories WHERE user_id = $1`
	_, err := s.pool.Exec(ctx, q, userID)
	if err != nil {
		return fmt.Errorf("delete repository: %w", err)
	}
	return nil
}
