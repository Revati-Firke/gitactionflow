import type { ConnectedRepo, ListedRepo } from '../../types'
import { EmptyState } from '../EmptyState/EmptyState'

type RepositoryCardProps = {
  connected: ConnectedRepo | null
  candidates: ListedRepo[]
  loading: boolean
  connecting: boolean
  picking: boolean
  onStartPick: () => void
  onCancelPick: () => void
  onConnect: (githubId: number) => void
  onDisconnect: () => void
  error: string | null
}

export function RepositoryCard({
  connected,
  candidates,
  loading,
  connecting,
  picking,
  onStartPick,
  onCancelPick,
  onConnect,
  onDisconnect,
  error,
}: RepositoryCardProps) {
  if (loading) {
    return (
      <section className="card" aria-busy="true">
        <h2>Connected repository</h2>
        <p className="muted">Loading repository…</p>
      </section>
    )
  }

  if (connected && !picking) {
    return (
      <section className="card">
        <div className="card-head">
          <h2>Connected repository</h2>
          <button
            type="button"
            className="btn danger"
            disabled={connecting}
            onClick={() => {
              if (window.confirm('Disconnect this repository?\n\nYour GitActionFlow configuration for this repository will no longer be active.')) {
                onDisconnect()
              }
            }}
          >
            Disconnect
          </button>
        </div>
        {error && <p className="error" role="alert">{error}</p>}
        <p className="repo-full">{connected.full_name}</p>
        <dl className="meta-grid">
          <div>
            <dt>Visibility</dt>
            <dd>{connected.private ? 'Private' : 'Public'}</dd>
          </div>
          <div>
            <dt>Default branch</dt>
            <dd>{connected.default_branch}</dd>
          </div>
          <div>
            <dt>Owner</dt>
            <dd>{connected.owner_login}</dd>
          </div>
        </dl>
        <a className="link" href={connected.html_url} target="_blank" rel="noreferrer">
          Open on GitHub
        </a>
        <p className="note">Only one repository can be connected. Disconnect to switch.</p>
      </section>
    )
  }

  return (
    <section className="card">
      <div className="card-head">
        <h2>{picking ? 'Select a repository' : 'Connected repository'}</h2>
        {picking && (
          <button type="button" className="btn ghost" onClick={onCancelPick} disabled={connecting}>
            Cancel
          </button>
        )}
      </div>
      {error && <p className="error" role="alert">{error}</p>}
      {!picking ? (
        <EmptyState
          title="No repository connected"
          description="Connect one GitHub repository to start GitActionFlow automation."
          action={
            <button type="button" className="btn" onClick={onStartPick} disabled={connecting}>
              Connect repository
            </button>
          }
        />
      ) : candidates.length === 0 ? (
        <p className="muted">No repositories found (admin access required).</p>
      ) : (
        <ul className="repo-pick">
          {candidates.map((repo) => (
            <li key={repo.id}>
              <div>
                <p className="repo-full">{repo.full_name}</p>
                <p className="muted">
                  {repo.private ? 'Private' : 'Public'} · {repo.default_branch}
                </p>
              </div>
              <button
                type="button"
                className="btn"
                disabled={connecting}
                onClick={() => onConnect(repo.id)}
              >
                Connect
              </button>
            </li>
          ))}
        </ul>
      )}
    </section>
  )
}
