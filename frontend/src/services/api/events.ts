import type { WebhookEventSummary } from '../../types'
import { apiFetch } from './client'

export async function listEvents(page = 1, limit = 20): Promise<WebhookEventSummary[]> {
  const body = await apiFetch<{ events: WebhookEventSummary[] }>(
    `/api/events?page=${page}&limit=${limit}`,
  )
  return body.events
}
