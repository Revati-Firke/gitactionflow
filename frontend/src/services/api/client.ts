import type { ApiError } from '../../types'

// Must match the OAuth callback host (localhost, not 127.0.0.1).
// Cookies are host-bound: a session set on localhost:8080 is sent only to that host.
const API_BASE =
  (import.meta.env.VITE_API_BASE_URL as string | undefined)?.replace(/\/$/, '') ??
  'http://localhost:8080'

export class ApiRequestError extends Error {
  code?: string
  status: number

  constructor(status: number, err: ApiError) {
    super(err.message)
    this.name = 'ApiRequestError'
    this.status = status
    this.code = err.code
  }
}

function url(path: string): string {
  return `${API_BASE}${path}`
}

async function parseBody(res: Response): Promise<unknown> {
  const text = await res.text()
  if (!text) return null
  try {
    return JSON.parse(text) as unknown
  } catch {
    return null
  }
}

export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url(path), {
    credentials: 'include',
    ...init,
    headers: {
      ...(init?.body ? { 'Content-Type': 'application/json' } : {}),
      ...init?.headers,
    },
  })
  const body = await parseBody(res)
  if (!res.ok) {
    const err = (body as { error?: ApiError } | null)?.error
    throw new ApiRequestError(res.status, {
      code: err?.code,
      message: err?.message ?? `request failed (${res.status})`,
    })
  }
  return body as T
}

export function loginURL(): string {
  return url('/auth/github')
}
