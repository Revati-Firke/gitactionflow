package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Revati-Firke/gitactionflow/backend/internal/auth"
)

// Users persists application users.
type Users struct {
	pool *pgxpool.Pool
}

func NewUsers(pool *pgxpool.Pool) *Users {
	return &Users{pool: pool}
}

// UpsertFromGitHub creates or updates a user by GitHub user ID and stores the encrypted token.
func (s *Users) UpsertFromGitHub(ctx context.Context, githubUserID int64, login, name, avatarURL string, encryptedToken []byte) (auth.User, error) {
	const q = `
INSERT INTO users (
    github_user_id, github_username, display_name, avatar_url, github_access_token_encrypted
) VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (github_user_id) DO UPDATE SET
    github_username = EXCLUDED.github_username,
    display_name = EXCLUDED.display_name,
    avatar_url = EXCLUDED.avatar_url,
    github_access_token_encrypted = EXCLUDED.github_access_token_encrypted,
    updated_at = NOW()
RETURNING id, github_user_id, github_username, display_name, avatar_url, created_at, updated_at
`
	var u auth.User
	err := s.pool.QueryRow(ctx, q, githubUserID, login, name, avatarURL, encryptedToken).Scan(
		&u.ID, &u.GitHubUserID, &u.GitHubUsername, &u.DisplayName, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return auth.User{}, fmt.Errorf("upsert user: %w", err)
	}
	return u, nil
}

// GetByID loads a user by internal ID.
func (s *Users) GetByID(ctx context.Context, id uuid.UUID) (auth.User, error) {
	const q = `
SELECT id, github_user_id, github_username, display_name, avatar_url, created_at, updated_at
FROM users WHERE id = $1
`
	var u auth.User
	err := s.pool.QueryRow(ctx, q, id).Scan(
		&u.ID, &u.GitHubUserID, &u.GitHubUsername, &u.DisplayName, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.User{}, ErrNotFound
	}
	if err != nil {
		return auth.User{}, fmt.Errorf("get user: %w", err)
	}
	return u, nil
}

// GetEncryptedAccessToken returns the encrypted GitHub token bytes for a user.
func (s *Users) GetEncryptedAccessToken(ctx context.Context, id uuid.UUID) ([]byte, error) {
	const q = `SELECT github_access_token_encrypted FROM users WHERE id = $1`
	var enc []byte
	err := s.pool.QueryRow(ctx, q, id).Scan(&enc)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}
	if len(enc) == 0 {
		return nil, ErrNotFound
	}
	return enc, nil
}
