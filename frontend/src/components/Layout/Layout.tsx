import type { ReactNode } from 'react'
import type { User } from '../../types'

type LayoutProps = {
  user: User | null
  onLogout?: () => void
  children: ReactNode
}

export function Layout({ user, onLogout, children }: LayoutProps) {
  return (
    <div className="shell">
      <header className="topbar">
        <div className="brand">
          <span className="brand-mark" aria-hidden="true" />
          <span className="brand-name">GitActionFlow</span>
        </div>
        {user && (
          <div className="topbar-actions">
            <span className="user-chip">
              {user.avatar_url ? (
                <img src={user.avatar_url} alt="" width={24} height={24} className="avatar" />
              ) : null}
              <span>{user.github_username}</span>
            </span>
            <button type="button" className="btn ghost" onClick={onLogout}>
              Log out
            </button>
          </div>
        )}
      </header>
      <main className="content">{children}</main>
    </div>
  )
}
