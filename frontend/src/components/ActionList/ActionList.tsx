import type { ActionSummary } from '../../types'
import { EmptyState } from '../EmptyState/EmptyState'
import { StatusBadge } from '../StatusBadge/StatusBadge'

function formatWhen(iso: string): string {
  try {
    return new Date(iso).toLocaleString()
  } catch {
    return iso
  }
}

function actionLabel(t: string): string {
  if (t === 'github_label') return 'GitHub label'
  if (t === 'github_comment') return 'GitHub comment'
  if (t === 'slack_notification') return 'Slack notification'
  return t
}

type ActionListProps = {
  actions: ActionSummary[]
  loading: boolean
  error: string | null
  repoConnected: boolean
  onRefresh: () => void
}

export function ActionList({ actions, loading, error, repoConnected, onRefresh }: ActionListProps) {
  return (
    <section className="card">
      <div className="card-head">
        <h2>Recent actions</h2>
        <button type="button" className="btn ghost" onClick={onRefresh} disabled={!repoConnected || loading}>
          Refresh
        </button>
      </div>

      {!repoConnected ? (
        <EmptyState title="No actions yet" description="Actions appear after rules match webhook events." />
      ) : loading ? (
        <p className="muted">Loading actions…</p>
      ) : error ? (
        <div className="error-block">
          <p className="error" role="alert">
            Unable to load actions.
          </p>
          <button type="button" className="btn ghost" onClick={onRefresh}>
            Try again
          </button>
        </div>
      ) : actions.length === 0 ? (
        <EmptyState
          title="No actions have been executed yet"
          description="Matching rules create durable actions that call GitHub or Slack."
        />
      ) : (
        <ul className="action-list">
          {actions.map((a) => (
            <li key={a.id} className="action-item">
              <div>
                <div className="rule-title-row">
                  <strong>{actionLabel(a.action_type)}</strong>
                  <StatusBadge status={a.status} />
                </div>
                <p className="muted">
                  Rule: {a.rule_name ?? '(deleted or unknown)'} · Attempts: {a.attempt_count}
                  {a.max_attempts ? ` / ${a.max_attempts}` : ''}
                </p>
                <p className="muted">Created {formatWhen(a.created_at)}</p>
                {a.last_error && (
                  <p className="error" role="alert">
                    {a.last_error}
                  </p>
                )}
              </div>
            </li>
          ))}
        </ul>
      )}
    </section>
  )
}
