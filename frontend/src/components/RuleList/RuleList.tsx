import type { Rule, RuleInput } from '../../types'
import { EmptyState } from '../EmptyState/EmptyState'
import { StatusBadge } from '../StatusBadge/StatusBadge'
import { RuleForm } from '../RuleForm/RuleForm'

function actionSummary(rule: Rule): string {
  const cfg = rule.action_config
  if (rule.action_type === 'github_label') return `GitHub label → ${String(cfg.label ?? '')}`
  if (rule.action_type === 'github_comment') return 'GitHub comment'
  if (rule.action_type === 'slack_notification') return 'Slack notification'
  return rule.action_type
}

type RuleListProps = {
  rules: Rule[]
  loading: boolean
  error: string | null
  formOpen: boolean
  editing: Rule | null
  formBusy: boolean
  formError: string | null
  repoConnected: boolean
  onOpenCreate: () => void
  onEdit: (rule: Rule) => void
  onCancelForm: () => void
  onSubmit: (input: RuleInput) => Promise<void>
  onToggle: (rule: Rule) => void
  onDelete: (rule: Rule) => void
  onRetry: () => void
}

export function RuleList(props: RuleListProps) {
  const {
    rules,
    loading,
    error,
    formOpen,
    editing,
    formBusy,
    formError,
    repoConnected,
    onOpenCreate,
    onEdit,
    onCancelForm,
    onSubmit,
    onToggle,
    onDelete,
    onRetry,
  } = props

  return (
    <section className="card">
      <div className="card-head">
        <h2>Automation rules</h2>
        {repoConnected && !formOpen && (
          <button type="button" className="btn" onClick={onOpenCreate}>
            Create rule
          </button>
        )}
      </div>

      {!repoConnected ? (
        <EmptyState
          title="Connect a repository first"
          description="Rules apply to your one connected GitHub repository."
        />
      ) : loading ? (
        <p className="muted">Loading rules…</p>
      ) : error ? (
        <div className="error-block">
          <p className="error" role="alert">
            Unable to load automation rules.
          </p>
          <button type="button" className="btn ghost" onClick={onRetry}>
            Try again
          </button>
        </div>
      ) : (
        <>
          {formOpen && (
            <RuleForm
              initial={editing}
              busy={formBusy}
              error={formError}
              onSubmit={onSubmit}
              onCancel={onCancelForm}
            />
          )}
          {rules.length === 0 && !formOpen ? (
            <EmptyState
              title="No automation rules yet"
              description="Create your first rule to automate GitHub events."
              action={
                <button type="button" className="btn" onClick={onOpenCreate}>
                  Create rule
                </button>
              }
            />
          ) : (
            <ul className="rule-list">
              {rules.map((rule) => (
                <li key={rule.id} className="rule-item">
                  <div className="rule-main">
                    <div className="rule-title-row">
                      <h3>{rule.name}</h3>
                      <StatusBadge status={rule.enabled ? 'enabled' : 'disabled'} />
                    </div>
                    <p className="muted">
                      Event: {rule.event_type === 'pull_request' ? 'Pull request' : 'Issue'}
                      {rule.keyword ? ` · Keyword: ${rule.keyword}` : ''}
                      {rule.author ? ` · Author: ${rule.author}` : ''}
                      {rule.required_labels?.length
                        ? ` · Labels: ${rule.required_labels.join(', ')}`
                        : ''}
                    </p>
                    <p>{actionSummary(rule)}</p>
                  </div>
                  <div className="rule-actions">
                    <label className="switch">
                      <span className="sr-only">Enable {rule.name}</span>
                      <input
                        type="checkbox"
                        checked={rule.enabled}
                        onChange={() => onToggle(rule)}
                      />
                      <span>{rule.enabled ? 'On' : 'Off'}</span>
                    </label>
                    <button type="button" className="btn ghost" onClick={() => onEdit(rule)}>
                      Edit
                    </button>
                    <button
                      type="button"
                      className="btn ghost"
                      onClick={() => {
                        if (window.confirm('Delete this rule?\n\nPast events and actions remain visible.')) {
                          onDelete(rule)
                        }
                      }}
                    >
                      Delete
                    </button>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </>
      )}
    </section>
  )
}
