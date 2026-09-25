import type { WebhookEventSummary } from '../../types'
import { EmptyState } from '../EmptyState/EmptyState'
import { StatusBadge } from '../StatusBadge/StatusBadge'

function formatWhen(iso: string): string {
  try {
    const d = new Date(iso)
    return d.toLocaleString()
  } catch {
    return iso
  }
}

function eventLabel(t: string): string {
  if (t === 'pull_request') return 'Pull request'
  if (t === 'issues') return 'Issue'
  return t
}

type EventListProps = {
  events: WebhookEventSummary[]
  loading: boolean
  error: string | null
  repoConnected: boolean
  selectedId: string | null
  onSelect: (id: string | null) => void
  onRefresh: () => void
}

export function EventList({
  events,
  loading,
  error,
  repoConnected,
  selectedId,
  onSelect,
  onRefresh,
}: EventListProps) {
  const selected = events.find((e) => e.id === selectedId) ?? null

  return (
    <section className="card">
      <div className="card-head">
        <h2>Recent events</h2>
        <button type="button" className="btn ghost" onClick={onRefresh} disabled={!repoConnected || loading}>
          Refresh
        </button>
      </div>

      {!repoConnected ? (
        <EmptyState title="No events yet" description="Connect a repository to receive GitHub webhooks." />
      ) : loading ? (
        <p className="muted">Loading events…</p>
      ) : error ? (
        <div className="error-block">
          <p className="error" role="alert">
            Unable to load events.
          </p>
          <button type="button" className="btn ghost" onClick={onRefresh}>
            Try again
          </button>
        </div>
      ) : events.length === 0 ? (
        <EmptyState
          title="No GitHub events have been received yet"
          description="Open or update an issue/PR on the connected repository to generate activity."
        />
      ) : (
        <>
          <div className="table-wrap">
            <table className="data-table">
              <thead>
                <tr>
                  <th scope="col">Event</th>
                  <th scope="col">Action</th>
                  <th scope="col">Status</th>
                  <th scope="col">Received</th>
                </tr>
              </thead>
              <tbody>
                {events.map((ev) => (
                  <tr
                    key={ev.id}
                    className={selectedId === ev.id ? 'selected' : undefined}
                    onClick={() => onSelect(selectedId === ev.id ? null : ev.id)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter' || e.key === ' ') {
                        e.preventDefault()
                        onSelect(selectedId === ev.id ? null : ev.id)
                      }
                    }}
                    tabIndex={0}
                    role="button"
                    aria-pressed={selectedId === ev.id}
                  >
                    <td>{eventLabel(ev.event_type)}</td>
                    <td>{ev.action}</td>
                    <td>
                      <StatusBadge status={ev.status} />
                    </td>
                    <td>{formatWhen(ev.received_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {selected && (
            <div className="detail">
              <h3>
                {eventLabel(selected.event_type)} · {selected.action}
              </h3>
              <dl className="meta-grid">
                <div>
                  <dt>Status</dt>
                  <dd>
                    <StatusBadge status={selected.status} />
                  </dd>
                </div>
                <div>
                  <dt>Received</dt>
                  <dd>{formatWhen(selected.received_at)}</dd>
                </div>
                {selected.processed_at && (
                  <div>
                    <dt>Processed</dt>
                    <dd>{formatWhen(selected.processed_at)}</dd>
                  </div>
                )}
                <div>
                  <dt>Delivery ID</dt>
                  <dd className="mono">{selected.delivery_id}</dd>
                </div>
                <div>
                  <dt>Retries</dt>
                  <dd>{selected.retry_count}</dd>
                </div>
              </dl>
              {selected.last_error && (
                <p className="error" role="alert">
                  {selected.last_error}
                </p>
              )}
            </div>
          )}
        </>
      )}
    </section>
  )
}
