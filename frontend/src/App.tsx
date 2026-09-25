import { useCallback, useEffect, useState } from 'react'
import { Layout } from './components/Layout/Layout'
import { DashboardPage } from './pages/DashboardPage'
import { LoginPage } from './pages/LoginPage'
import { getMe, logout } from './services/api'
import type { User } from './types'

function navigate(to: string) {
  if (window.location.pathname !== to) {
    window.history.pushState({}, '', to)
  }
  window.dispatchEvent(new PopStateEvent('popstate'))
}

function usePath(): string {
  const [path, setPath] = useState(() => window.location.pathname)
  useEffect(() => {
    const onPop = () => setPath(window.location.pathname)
    window.addEventListener('popstate', onPop)
    return () => window.removeEventListener('popstate', onPop)
  }, [])
  return path
}

export default function App() {
  const path = usePath()
  const [user, setUser] = useState<User | null | undefined>(undefined)
  const [bootError, setBootError] = useState<string | null>(null)

  const refreshSession = useCallback(async () => {
    setBootError(null)
    try {
      const me = await getMe()
      setUser(me)
      if (me) {
        if (path === '/' || path === '/login') navigate('/dashboard')
      } else if (path === '/dashboard') {
        navigate('/login')
      } else if (path === '/') {
        navigate('/login')
      }
    } catch (err) {
      setUser(null)
      setBootError(err instanceof Error ? err.message : 'failed to load session')
      navigate('/login')
    }
  }, [path])

  useEffect(() => {
    void refreshSession()
    // Intentionally run once on mount for session bootstrap.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    if (user === undefined) return
    if (user && (path === '/' || path === '/login')) navigate('/dashboard')
    if (!user && path === '/dashboard') navigate('/login')
  }, [user, path])

  async function onLogout() {
    await logout()
    setUser(null)
    navigate('/login')
  }

  if (user === undefined) {
    return (
      <Layout user={null}>
        <p className="muted loading-screen">Checking session…</p>
      </Layout>
    )
  }

  if (!user) {
    return (
      <Layout user={null}>
        <LoginPage error={bootError} />
      </Layout>
    )
  }

  return (
    <Layout user={user} onLogout={() => void onLogout()}>
      <DashboardPage user={user} />
    </Layout>
  )
}
