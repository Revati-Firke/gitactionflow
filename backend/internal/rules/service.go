package rules

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/Revati-Firke/gitactionflow/backend/internal/store"
)

var (
	ErrNoRepository = errors.New("no repository connected")
	ErrNotFound     = errors.New("rule not found")
)

// RuleStore is the persistence surface used by the service.
type RuleStore interface {
	Create(ctx context.Context, r store.Rule) (store.Rule, error)
	ListByRepository(ctx context.Context, repositoryID uuid.UUID) ([]store.Rule, error)
	ListEnabledByRepository(ctx context.Context, repositoryID uuid.UUID) ([]store.Rule, error)
	GetByID(ctx context.Context, id uuid.UUID) (store.Rule, error)
	Update(ctx context.Context, repositoryID, id uuid.UUID, r store.Rule) (store.Rule, error)
	Delete(ctx context.Context, repositoryID, id uuid.UUID) error
}

// RepoStore resolves the authenticated user's connected repository.
type RepoStore interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (store.Repository, error)
}

// Service manages rules for the caller's connected repository.
type Service struct {
	Rules RuleStore
	Repos  RepoStore
}

func (s *Service) connectedRepo(ctx context.Context, userID uuid.UUID) (store.Repository, error) {
	repo, err := s.Repos.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return store.Repository{}, ErrNoRepository
		}
		return store.Repository{}, err
	}
	return repo, nil
}

// Create adds a rule for the user's connected repository.
func (s *Service) Create(ctx context.Context, userID uuid.UUID, in Input) (store.Rule, error) {
	norm, err := ValidateNormalize(in)
	if err != nil {
		return store.Rule{}, err
	}
	repo, err := s.connectedRepo(ctx, userID)
	if err != nil {
		return store.Rule{}, err
	}
	return s.Rules.Create(ctx, store.Rule{
		RepositoryID:   repo.ID,
		Name:           norm.Name,
		Enabled:        norm.Enabled,
		EventType:      norm.EventType,
		Keyword:        norm.Keyword,
		Author:         norm.Author,
		RequiredLabels: norm.RequiredLabels,
		ActionType:     norm.ActionType,
		ActionConfig:   norm.ActionConfig,
	})
}

// List returns all rules for the connected repository.
func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]store.Rule, error) {
	repo, err := s.connectedRepo(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.Rules.ListByRepository(ctx, repo.ID)
}

// Get returns one rule if it belongs to the connected repository.
func (s *Service) Get(ctx context.Context, userID, ruleID uuid.UUID) (store.Rule, error) {
	repo, err := s.connectedRepo(ctx, userID)
	if err != nil {
		return store.Rule{}, err
	}
	r, err := s.Rules.GetByID(ctx, ruleID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return store.Rule{}, ErrNotFound
		}
		return store.Rule{}, err
	}
	if r.RepositoryID != repo.ID {
		return store.Rule{}, ErrNotFound
	}
	return r, nil
}

// Update replaces a rule owned by the connected repository.
func (s *Service) Update(ctx context.Context, userID, ruleID uuid.UUID, in Input) (store.Rule, error) {
	norm, err := ValidateNormalize(in)
	if err != nil {
		return store.Rule{}, err
	}
	repo, err := s.connectedRepo(ctx, userID)
	if err != nil {
		return store.Rule{}, err
	}
	updated, err := s.Rules.Update(ctx, repo.ID, ruleID, store.Rule{
		Name:           norm.Name,
		Enabled:        norm.Enabled,
		EventType:      norm.EventType,
		Keyword:        norm.Keyword,
		Author:         norm.Author,
		RequiredLabels: norm.RequiredLabels,
		ActionType:     norm.ActionType,
		ActionConfig:   norm.ActionConfig,
	})
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return store.Rule{}, ErrNotFound
		}
		return store.Rule{}, err
	}
	return updated, nil
}

// Delete removes a rule owned by the connected repository.
func (s *Service) Delete(ctx context.Context, userID, ruleID uuid.UUID) error {
	repo, err := s.connectedRepo(ctx, userID)
	if err != nil {
		return err
	}
	if err := s.Rules.Delete(ctx, repo.ID, ruleID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}