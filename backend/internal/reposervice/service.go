package reposervice

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/Revati-Firke/gitactionflow/backend/internal/auth"
	"github.com/Revati-Firke/gitactionflow/backend/internal/githubapi"
	"github.com/Revati-Firke/gitactionflow/backend/internal/store"
)

var (
	ErrAlreadyConnected   = errors.New("repository already connected")
	ErrAccessDenied       = errors.New("repository access denied")
	ErrRepositoryNotFound = errors.New("repository not found")
	ErrGitHubUnauthorized = errors.New("github unauthorized")
	ErrGitHubRateLimited  = errors.New("github rate limited")
	ErrInvalidInput       = errors.New("invalid input")
	ErrTokenUnavailable   = errors.New("github token unavailable")
)

// GitHubClient is the GitHub API surface used by this service.
type GitHubClient interface {
	ListRepositories(ctx context.Context, accessToken string) ([]githubapi.Repository, error)
	GetRepository(ctx context.Context, accessToken string, githubRepoID int64) (githubapi.Repository, error)
}

// UserTokens loads and decrypts GitHub access tokens.
type UserTokens interface {
	GetEncryptedAccessToken(ctx context.Context, id uuid.UUID) ([]byte, error)
}

// RepoStore persists connected repositories.
type RepoStore interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (store.Repository, error)
	Insert(ctx context.Context, userID uuid.UUID, githubRepoID int64, name, fullName, ownerLogin, defaultBranch, htmlURL string, private bool) (store.Repository, error)
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}

// Service implements repository-management application logic.
type Service struct {
	GitHub   GitHubClient
	Users    UserTokens
	Repos    RepoStore
	TokenKey []byte
}

// ListedRepo is a safe API DTO for GitHub listing.
type ListedRepo struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Private       bool   `json:"private"`
	DefaultBranch string `json:"default_branch"`
	HTMLURL       string `json:"html_url"`
	OwnerLogin    string `json:"owner_login"`
}

// ConnectedRepo is the persisted connection DTO.
type ConnectedRepo struct {
	ID                 string `json:"id"`
	GitHubRepositoryID int64  `json:"github_repository_id"`
	Name               string `json:"name"`
	FullName           string `json:"full_name"`
	OwnerLogin         string `json:"owner_login"`
	DefaultBranch      string `json:"default_branch"`
	HTMLURL            string `json:"html_url"`
	Private            bool   `json:"private"`
}

func toConnected(r store.Repository) ConnectedRepo {
	return ConnectedRepo{
		ID:                 r.ID.String(),
		GitHubRepositoryID: r.GitHubRepositoryID,
		Name:               r.Name,
		FullName:           r.FullName,
		OwnerLogin:         r.OwnerLogin,
		DefaultBranch:      r.DefaultBranch,
		HTMLURL:            r.HTMLURL,
		Private:            r.Private,
	}
}

func (s *Service) accessToken(ctx context.Context, userID uuid.UUID) (string, error) {
	enc, err := s.Users.GetEncryptedAccessToken(ctx, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return "", ErrTokenUnavailable
		}
		return "", err
	}
	plain, err := auth.Decrypt(s.TokenKey, enc)
	if err != nil {
		return "", ErrTokenUnavailable
	}
	return string(plain), nil
}

func mapGitHubErr(err error) error {
	switch {
	case errors.Is(err, githubapi.ErrUnauthorized):
		return ErrGitHubUnauthorized
	case errors.Is(err, githubapi.ErrRateLimited):
		return ErrGitHubRateLimited
	case errors.Is(err, githubapi.ErrNotFound):
		return ErrRepositoryNotFound
	case errors.Is(err, githubapi.ErrForbidden):
		return ErrAccessDenied
	default:
		return err
	}
}

// ListGitHubRepositories returns repos visible to the user via GitHub.
func (s *Service) ListGitHubRepositories(ctx context.Context, userID uuid.UUID) ([]ListedRepo, error) {
	token, err := s.accessToken(ctx, userID)
	if err != nil {
		return nil, err
	}
	repos, err := s.GitHub.ListRepositories(ctx, token)
	if err != nil {
		return nil, mapGitHubErr(err)
	}
	out := make([]ListedRepo, 0, len(repos))
	for _, r := range repos {
		out = append(out, ListedRepo{
			ID:            r.ID,
			Name:          r.Name,
			FullName:      r.FullName,
			Private:       r.Private,
			DefaultBranch: r.DefaultBranch,
			HTMLURL:       r.HTMLURL,
			OwnerLogin:    r.Owner.Login,
		})
	}
	return out, nil
}

// GetConnected returns the user's connected repository.
func (s *Service) GetConnected(ctx context.Context, userID uuid.UUID) (ConnectedRepo, error) {
	r, err := s.Repos.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return ConnectedRepo{}, store.ErrNotFound
		}
		return ConnectedRepo{}, err
	}
	return toConnected(r), nil
}

// Connect validates access via GitHub and persists exactly one repository.
// Access rule: authenticated user must have GitHub admin permission on the repo
// (owners have admin; required later for webhook management).
func (s *Service) Connect(ctx context.Context, userID uuid.UUID, githubRepoID int64) (ConnectedRepo, error) {
	if githubRepoID <= 0 {
		return ConnectedRepo{}, ErrInvalidInput
	}

	if _, err := s.Repos.GetByUserID(ctx, userID); err == nil {
		return ConnectedRepo{}, ErrAlreadyConnected
	} else if !errors.Is(err, store.ErrNotFound) {
		return ConnectedRepo{}, err
	}

	token, err := s.accessToken(ctx, userID)
	if err != nil {
		return ConnectedRepo{}, err
	}

	ghRepo, err := s.GitHub.GetRepository(ctx, token, githubRepoID)
	if err != nil {
		return ConnectedRepo{}, mapGitHubErr(err)
	}

	if !ghRepo.Permissions.Admin {
		return ConnectedRepo{}, ErrAccessDenied
	}

	saved, err := s.Repos.Insert(
		ctx,
		userID,
		ghRepo.ID,
		ghRepo.Name,
		ghRepo.FullName,
		ghRepo.Owner.Login,
		ghRepo.DefaultBranch,
		ghRepo.HTMLURL,
		ghRepo.Private,
	)
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			return ConnectedRepo{}, ErrAlreadyConnected
		}
		return ConnectedRepo{}, fmt.Errorf("persist repository: %w", err)
	}
	return toConnected(saved), nil
}

// Disconnect removes the local connection (idempotent). No webhook cleanup in Phase 4.
func (s *Service) Disconnect(ctx context.Context, userID uuid.UUID) error {
	return s.Repos.DeleteByUserID(ctx, userID)
}
