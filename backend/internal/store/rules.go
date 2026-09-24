package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Rule is a user-configured automation rule for one connected repository.
type Rule struct {
	ID              uuid.UUID
	RepositoryID    uuid.UUID
	Name            string
	Enabled         bool
	EventType       string
	Keyword         *string
	Author          *string
	RequiredLabels  []string
	ActionType      string
	ActionConfig    json.RawMessage
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Rules persists rule configuration.
type Rules struct {
	pool *pgxpool.Pool
}

func NewRules(pool *pgxpool.Pool) *Rules {
	return &Rules{pool: pool}
}

const ruleColumns = `
id, repository_id, name, enabled, event_type, keyword, author, required_labels,
action_type, action_config, created_at, updated_at`

func scanRule(row pgx.Row) (Rule, error) {
	var r Rule
	err := row.Scan(
		&r.ID, &r.RepositoryID, &r.Name, &r.Enabled, &r.EventType, &r.Keyword, &r.Author, &r.RequiredLabels,
		&r.ActionType, &r.ActionConfig, &r.CreatedAt, &r.UpdatedAt,
	)
	if r.RequiredLabels == nil {
		r.RequiredLabels = []string{}
	}
	return r, err
}

// Create inserts a new rule.
func (s *Rules) Create(ctx context.Context, r Rule) (Rule, error) {
	const q = `
INSERT INTO rules (
    repository_id, name, enabled, event_type, keyword, author, required_labels, action_type, action_config
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING ` + ruleColumns
	labels := r.RequiredLabels
	if labels == nil {
		labels = []string{}
	}
	out, err := scanRule(s.pool.QueryRow(ctx, q,
		r.RepositoryID, r.Name, r.Enabled, r.EventType, r.Keyword, r.Author, labels, r.ActionType, r.ActionConfig,
	))
	if err != nil {
		return Rule{}, fmt.Errorf("create rule: %w", err)
	}
	return out, nil
}

// ListByRepository returns all rules for a repository (enabled and disabled), oldest first.
func (s *Rules) ListByRepository(ctx context.Context, repositoryID uuid.UUID) ([]Rule, error) {
	const q = `SELECT ` + ruleColumns + ` FROM rules WHERE repository_id = $1 ORDER BY created_at ASC, id ASC`
	rows, err := s.pool.Query(ctx, q, repositoryID)
	if err != nil {
		return nil, fmt.Errorf("list rules: %w", err)
	}
	defer rows.Close()
	var out []Rule
	for rows.Next() {
		r, err := scanRule(rows)
		if err != nil {
			return nil, fmt.Errorf("scan rule: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListEnabledByRepository returns enabled rules for evaluation, deterministic order.
func (s *Rules) ListEnabledByRepository(ctx context.Context, repositoryID uuid.UUID) ([]Rule, error) {
	const q = `
SELECT ` + ruleColumns + `
FROM rules
WHERE repository_id = $1 AND enabled = TRUE
ORDER BY created_at ASC, id ASC`
	rows, err := s.pool.Query(ctx, q, repositoryID)
	if err != nil {
		return nil, fmt.Errorf("list enabled rules: %w", err)
	}
	defer rows.Close()
	var out []Rule
	for rows.Next() {
		r, err := scanRule(rows)
		if err != nil {
			return nil, fmt.Errorf("scan rule: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetByID loads a rule by id.
func (s *Rules) GetByID(ctx context.Context, id uuid.UUID) (Rule, error) {
	const q = `SELECT ` + ruleColumns + ` FROM rules WHERE id = $1`
	r, err := scanRule(s.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Rule{}, ErrNotFound
	}
	if err != nil {
		return Rule{}, fmt.Errorf("get rule: %w", err)
	}
	return r, nil
}

// Update replaces mutable fields for a rule owned by repositoryID.
func (s *Rules) Update(ctx context.Context, repositoryID, id uuid.UUID, r Rule) (Rule, error) {
	const q = `
UPDATE rules SET
    name = $3,
    enabled = $4,
    event_type = $5,
    keyword = $6,
    author = $7,
    required_labels = $8,
    action_type = $9,
    action_config = $10,
    updated_at = NOW()
WHERE id = $1 AND repository_id = $2
RETURNING ` + ruleColumns
	labels := r.RequiredLabels
	if labels == nil {
		labels = []string{}
	}
	out, err := scanRule(s.pool.QueryRow(ctx, q,
		id, repositoryID, r.Name, r.Enabled, r.EventType, r.Keyword, r.Author, labels, r.ActionType, r.ActionConfig,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return Rule{}, ErrNotFound
	}
	if err != nil {
		return Rule{}, fmt.Errorf("update rule: %w", err)
	}
	return out, nil
}

// Delete removes a rule scoped to repositoryID.
func (s *Rules) Delete(ctx context.Context, repositoryID, id uuid.UUID) error {
	const q = `DELETE FROM rules WHERE id = $1 AND repository_id = $2`
	tag, err := s.pool.Exec(ctx, q, id, repositoryID)
	if err != nil {
		return fmt.Errorf("delete rule: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
