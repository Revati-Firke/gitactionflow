import type { ActionSummary } from '../../types'
import { apiFetch } from './client'

export async function listActions(page = 1, limit = 20): Promise<ActionSummary[]> {
  const body = await apiFetch<{ actions: ActionSummary[] }>(
    `/api/actions?page=${page}&limit=${limit}`,
  )
  return body.actions
}
