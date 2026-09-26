-- Phase 7: configurable rules (match only; no GitHub/Slack execution).

CREATE TABLE rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repository_id UUID NOT NULL REFERENCES repositories (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    event_type TEXT NOT NULL,
    keyword TEXT,
    author TEXT,
    required_labels TEXT[] NOT NULL DEFAULT '{}',
    action_type TEXT NOT NULL,
    action_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT rules_event_type_check CHECK (event_type IN ('issues', 'pull_request')),
    CONSTRAINT rules_action_type_check CHECK (action_type IN ('github_label', 'github_comment', 'slack_notification')),
    CONSTRAINT rules_name_not_blank CHECK (length(trim(name)) > 0)
);

CREATE INDEX rules_repository_id_idx ON rules (repository_id);
CREATE INDEX rules_repository_enabled_idx ON rules (repository_id, enabled);
