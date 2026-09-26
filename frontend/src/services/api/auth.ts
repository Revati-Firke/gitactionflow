import type { User } from '../../types'
import { ApiRequestError, apiFetch, loginURL } from './client'

export { loginURL }

export async function getMe(): Promise<User | null> {
  try {
    const body = await apiFetch<{ user: User }>('/api/me')
    return body.user
  } catch (err) {
    if (err instanceof ApiRequestError && err.status === 401) return null
    throw err
  }
}

export async function logout(): Promise<void> {
  await apiFetch<{ status: string }>('/auth/logout', { method: 'POST' })
}
