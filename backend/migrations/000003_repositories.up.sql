-- Phase 4: one connected GitHub repository per user.

CREATE TABLE repositories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    github_repository_id BIGINT NOT NULL,
    name TEXT NOT NULL,
    full_name TEXT NOT NULL,
    owner_login TEXT NOT NULL,
    default_branch TEXT NOT NULL DEFAULT 'main',
    html_url TEXT NOT NULL DEFAULT '',
    private BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT repositories_user_id_unique UNIQUE (user_id),
    CONSTRAINT repositories_user_github_repo_unique UNIQUE (user_id, github_repository_id)
);

CREATE INDEX repositories_github_repository_id_idx ON repositories (github_repository_id);
