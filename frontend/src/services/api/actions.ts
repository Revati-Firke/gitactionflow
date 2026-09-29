import type { ActionSummary } from '../../types'
import { apiFetch } from './client'

export async function listActions(
  page = 1,
  limit = 20,
  eventType?: string,
): Promise<ActionSummary[]> {
  const params = new URLSearchParams({
    page: String(page),
    limit: String(limit),
  })
  if (eventType) {
    params.set('event_type', eventType)
  }
  const body = await apiFetch<{ actions: ActionSummary[] }>(`/api/actions?${params}`)
  return body.actions
}
