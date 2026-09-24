package auth

import (
	"time"

	"github.com/google/uuid"
)

// User is the authenticated application user (safe for API responses).
type User struct {
	ID             uuid.UUID
	GitHubUserID   int64
	GitHubUsername string
	DisplayName    string
	AvatarURL      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Session is a server-side session record (token hash only).
type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash []byte
	ExpiresAt time.Time
	CreatedAt time.Time
}

// PublicUser is the JSON shape returned by /api/me.
type PublicUser struct {
	ID             string `json:"id"`
	GitHubUsername string `json:"github_username"`
	DisplayName    string `json:"display_name"`
	AvatarURL      string `json:"avatar_url"`
}

// ToPublic maps a User to the API response shape.
func (u User) ToPublic() PublicUser {
	return PublicUser{
		ID:             u.ID.String(),
		GitHubUsername: u.GitHubUsername,
		DisplayName:    u.DisplayName,
		AvatarURL:      u.AvatarURL,
	}
}
