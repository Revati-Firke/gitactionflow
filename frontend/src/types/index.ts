export type User = {
  id: string
  github_username: string
  display_name: string
  avatar_url: string
}

export type ListedRepo = {
  id: number
  name: string
  full_name: string
  private: boolean
  default_branch: string
  html_url: string
  owner_login: string
}

export type ConnectedRepo = {
  id: string
  github_repository_id: number
  name: string
  full_name: string
  owner_login: string
  default_branch: string
  html_url: string
  private: boolean
}

export type Rule = {
  id: string
  name: string
  enabled: boolean
  event_type: 'issues' | 'pull_request' | string
  keyword: string | null
  author: string | null
  required_labels: string[]
  action_type: 'github_label' | 'github_comment' | 'slack_notification' | string
  action_config: Record<string, unknown>
  created_at: string
  updated_at: string
}

export type RuleInput = {
  name: string
  enabled: boolean
  event_type: string
  keyword?: string | null
  author?: string | null
  required_labels: string[]
  action_type: string
  action_config: Record<string, unknown>
}

export type WebhookEventSummary = {
  id: string
  event_type: string
  action: string
  status: string
  delivery_id: string
  retry_count: number
  last_error?: string | null
  received_at: string
  processed_at?: string | null
  failed_at?: string | null
}

export type ActionSummary = {
  id: string
  event_id: string
  rule_id?: string | null
  rule_name?: string | null
  action_type: string
  status: string
  attempt_count: number
  max_attempts: number
  last_error?: string | null
  created_at: string
  completed_at?: string | null
  failed_at?: string | null
}

export type ApiError = {
  code?: string
  message: string
}
