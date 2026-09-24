-- Phase 2 foundation: minimal metadata table to prove migrations work.
-- Full application schema (users, repos, rules, events, actions) lands in later phases.

CREATE TABLE IF NOT EXISTS app_meta (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO app_meta (key, value)
VALUES ('phase', '2_foundation')
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    updated_at = NOW();
