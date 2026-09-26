package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Revati-Firke/gitactionflow/backend/internal/auth"
)

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("conflict")

// Sessions persists server-side sessions (token hashes only).
type Sessions struct {
	pool *pgxpool.Pool
}

func NewSessions(pool *pgxpool.Pool) *Sessions {
	return &Sessions{pool: pool}
}

// Create inserts a new session.
func (s *Sessions) Create(ctx context.Context, userID uuid.UUID, tokenHash []byte, expiresAt time.Time) (auth.Session, error) {
	const q = `
INSERT INTO sessions (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING id, user_id, token_hash, expires_at, created_at
`
	var sess auth.Session
	err := s.pool.QueryRow(ctx, q, userID, tokenHash, expiresAt).Scan(
		&sess.ID, &sess.UserID, &sess.TokenHash, &sess.ExpiresAt, &sess.CreatedAt,
	)
	if err != nil {
		return auth.Session{}, fmt.Errorf("create session: %w", err)
	}
	return sess, nil
}

// GetValidByTokenHash returns a non-expired session.
func (s *Sessions) GetValidByTokenHash(ctx context.Context, tokenHash []byte, now time.Time) (auth.Session, error) {
	const q = `
SELECT id, user_id, token_hash, expires_at, created_at
FROM sessions
WHERE token_hash = $1 AND expires_at > $2
`
	var sess auth.Session
	err := s.pool.QueryRow(ctx, q, tokenHash, now).Scan(
		&sess.ID, &sess.UserID, &sess.TokenHash, &sess.ExpiresAt, &sess.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.Session{}, ErrNotFound
	}
	if err != nil {
		return auth.Session{}, fmt.Errorf("get session: %w", err)
	}
	return sess, nil
}

// DeleteByTokenHash removes a session. Missing rows are ignored (idempotent logout).
func (s *Sessions) DeleteByTokenHash(ctx context.Context, tokenHash []byte) error {
	const q = `DELETE FROM sessions WHERE token_hash = $1`
	_, err := s.pool.Exec(ctx, q, tokenHash)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}
