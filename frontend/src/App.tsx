import { useCallback, useEffect, useState } from 'react'
import {
  ConnectedRepo,
  ListedRepo,
  User,
  connectRepo,
  disconnectRepo,
  getConnectedRepo,
  getMe,
  listGitHubRepos,
  loginURL,
  logout,
} from './api'

export default function App() {
  const [user, setUser] = useState<User | null | undefined>(undefined)
  const [connected, setConnected] = useState<ConnectedRepo | null>(null)
  const [repos, setRepos] = useState<ListedRepo[]>([])
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  const refresh = useCallback(async () => {
    setError(null)
    const me = await getMe()
    setUser(me)
    if (!me) {
      setConnected(null)
      setRepos([])
      return
    }
    const current = await getConnectedRepo()
    setConnected(current)
    if (!current) {
      const listed = await listGitHubRepos()
      setRepos(listed)
    } else {
      setRepos([])
    }
  }, [])

  useEffect(() => {
    refresh().catch((err: unknown) => {
      setUser(null)
      setError(err instanceof Error ? err.message : 'failed to load')
    })
  }, [refresh])

  async function onConnect(id: number) {
    setBusy(true)
    setError(null)
    try {
      const repo = await connectRepo(id)
      setConnected(repo)
      setRepos([])
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'connect failed')
    } finally {
      setBusy(false)
    }
  }

  async function onDisconnect() {
    setBusy(true)
    setError(null)
    try {
      await disconnectRepo()
      await refresh()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'disconnect failed')
    } finally {
      setBusy(false)
    }
  }

  async function onLogout() {
    await logout()
    setUser(null)
    setConnected(null)
    setRepos([])
  }

  if (user === undefined) {
    return (
      <main className="page">
        <p className="muted">Loading…</p>
      </main>
    )
  }

  if (!user) {
    return (
      <main className="page">
        <h1>GitActionFlow</h1>
        <p className="lede">Connect one GitHub repository for event-driven automation.</p>
        {error && <p className="error">{error}</p>}
        <a className="button" href={loginURL()}>
          Sign in with GitHub
        </a>
        <p className="note">Webhook processing is not enabled yet (Phase 5).</p>
      </main>
    )
  }

  return (
    <main className="page">
      <header className="top">
        <div>
          <h1>GitActionFlow</h1>
          <p className="muted">
            Signed in as <strong>{user.github_username}</strong>
          </p>
        </div>
        <button type="button" className="button ghost" onClick={() => void onLogout()}>
          Log out
        </button>
      </header>

      {error && <p className="error">{error}</p>}

      {connected ? (
        <section className="panel">
          <h2>Connected repository</h2>
          <RepoDetails
            name={connected.name}
            owner={connected.owner_login}
            fullName={connected.full_name}
            privateRepo={connected.private}
            defaultBranch={connected.default_branch}
            htmlUrl={connected.html_url}
          />
          <button type="button" className="button danger" disabled={busy} onClick={() => void onDisconnect()}>
            Disconnect
          </button>
          <p className="note">Only one repository can be connected. Disconnect to choose another.</p>
        </section>
      ) : (
        <section className="panel">
          <h2>Select a repository</h2>
          <p className="muted">You need admin access on the repository. Listing comes from GitHub; nothing is saved until you connect.</p>
          {repos.length === 0 ? (
            <p className="muted">No repositories found for this account.</p>
          ) : (
            <ul className="repo-list">
              {repos.map((repo) => (
                <li key={repo.id} className="repo-row">
                  <RepoDetails
                    name={repo.name}
                    owner={repo.owner_login}
                    fullName={repo.full_name}
                    privateRepo={repo.private}
                    defaultBranch={repo.default_branch}
                    htmlUrl={repo.html_url}
                  />
                  <button type="button" className="button" disabled={busy} onClick={() => void onConnect(repo.id)}>
                    Connect
                  </button>
                </li>
              ))}
            </ul>
          )}
        </section>
      )}
    </main>
  )
}

function RepoDetails(props: {
  name: string
  owner: string
  fullName: string
  privateRepo: boolean
  defaultBranch: string
  htmlUrl: string
}) {
  return (
    <div className="repo-meta">
      <p className="repo-title">{props.fullName}</p>
      <p className="muted">
        Owner <strong>{props.owner}</strong> · {props.privateRepo ? 'Private' : 'Public'} · Branch{' '}
        <strong>{props.defaultBranch}</strong>
      </p>
      <a href={props.htmlUrl} target="_blank" rel="noreferrer">
        Open on GitHub
      </a>
    </div>
  )
}
