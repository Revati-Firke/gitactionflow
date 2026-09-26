import type { ApiError } from '../../types'

// Local: default to backend on localhost (use localhost, not 127.0.0.1 — cookies are host-bound).
// Production: default to same-origin "" so Vercel rewrites /api and /auth → Render, and the
// session cookie is first-party on the Vercel host (works when third-party cookies are blocked).
function resolveApiBase(): string {
  const raw = import.meta.env.VITE_API_BASE_URL
  if (typeof raw === 'string' && raw.trim() !== '') {
    return raw.replace(/\/$/, '')
  }
  if (import.meta.env.DEV) {
    return 'http://localhost:8080'
  }
  return ''
}

const API_BASE = resolveApiBase()

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
