package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OAuthStates stores one-time OAuth CSRF states (hashed).
type OAuthStates struct {
	pool *pgxpool.Pool
}

func NewOAuthStates(pool *pgxpool.Pool) *OAuthStates {
	return &OAuthStates{pool: pool}
}

// Create stores a new state hash with expiry.
func (s *OAuthStates) Create(ctx context.Context, stateHash []byte, expiresAt time.Time) error {
	const q = `
INSERT INTO oauth_states (state_hash, expires_at)
VALUES ($1, $2)
`
	_, err := s.pool.Exec(ctx, q, stateHash, expiresAt)
	if err != nil {
		return fmt.Errorf("create oauth state: %w", err)
	}
	return nil
}

// Consume validates and single-uses a state.
// Returns ErrNotFound if missing/expired/already used.
func (s *OAuthStates) Consume(ctx context.Context, stateHash []byte, now time.Time) error {
	const q = `
UPDATE oauth_states
SET consumed_at = $2
WHERE state_hash = $1
  AND consumed_at IS NULL
  AND expires_at > $2
RETURNING state_hash
`
	var out []byte
	err := s.pool.QueryRow(ctx, q, stateHash, now).Scan(&out)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return fmt.Errorf("consume oauth state: %w", err)
		}
		return fmt.Errorf("consume oauth state: %w", err)
	}
	return nil
}
