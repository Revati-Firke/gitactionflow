// Must match the OAuth callback host in .env (localhost, not 127.0.0.1).
// Cookies are host-bound: a session set on localhost is invisible to 127.0.0.1.
const API_BASE = (import.meta.env.VITE_API_BASE_URL as string | undefined)?.replace(/\/$/, '') ?? 'http://localhost:8080'

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

type ErrorBody = {
  error?: { code?: string; message?: string }
}

function url(path: string): string {
  return `${API_BASE}${path}`
}

async function parseJSON<T>(res: Response): Promise<T> {
  return (await res.json()) as T
}

export function loginURL(): string {
  return url('/auth/github')
}

export async function getMe(): Promise<User | null> {
  const res = await fetch(url('/api/me'), { credentials: 'include' })
  if (res.status === 401) return null
  if (!res.ok) {
    const body = await parseJSON<ErrorBody>(res)
    throw new Error(body.error?.message ?? 'failed to load user')
  }
  const body = await parseJSON<{ user: User }>(res)
  return body.user
}

export async function logout(): Promise<void> {
  await fetch(url('/auth/logout'), { method: 'POST', credentials: 'include' })
}

export async function listGitHubRepos(): Promise<ListedRepo[]> {
  const res = await fetch(url('/api/github/repositories'), { credentials: 'include' })
  if (!res.ok) {
    const body = await parseJSON<ErrorBody>(res)
    throw new Error(body.error?.message ?? 'failed to list repositories')
  }
  const body = await parseJSON<{ repositories: ListedRepo[] }>(res)
  return body.repositories
}

export async function getConnectedRepo(): Promise<ConnectedRepo | null> {
  const res = await fetch(url('/api/repository'), { credentials: 'include' })
  if (res.status === 404) return null
  if (!res.ok) {
    const body = await parseJSON<ErrorBody>(res)
    throw new Error(body.error?.message ?? 'failed to load connected repository')
  }
  const body = await parseJSON<{ repository: ConnectedRepo }>(res)
  return body.repository
}

export async function connectRepo(githubRepositoryId: number): Promise<ConnectedRepo> {
  const res = await fetch(url('/api/repository'), {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ github_repository_id: githubRepositoryId }),
  })
  if (!res.ok) {
    const body = await parseJSON<ErrorBody>(res)
    throw new Error(body.error?.message ?? 'failed to connect repository')
  }
  const body = await parseJSON<{ repository: ConnectedRepo }>(res)
  return body.repository
}

export async function disconnectRepo(): Promise<void> {
  const res = await fetch(url('/api/repository'), {
    method: 'DELETE',
    credentials: 'include',
  })
  if (!res.ok) {
    const body = await parseJSON<ErrorBody>(res)
    throw new Error(body.error?.message ?? 'failed to disconnect repository')
  }
}
