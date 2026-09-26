import type { Rule, RuleInput } from '../../types'
import { apiFetch } from './client'

export async function listRules(): Promise<Rule[]> {
  const body = await apiFetch<{ rules: Rule[] }>('/api/rules')
  return body.rules
}

export async function createRule(input: RuleInput): Promise<Rule> {
  const body = await apiFetch<{ rule: Rule }>('/api/rules', {
    method: 'POST',
    body: JSON.stringify(input),
  })
  return body.rule
}

export async function updateRule(id: string, input: RuleInput): Promise<Rule> {
  const body = await apiFetch<{ rule: Rule }>(`/api/rules/${id}`, {
    method: 'PUT',
    body: JSON.stringify(input),
  })
  return body.rule
}

export async function deleteRule(id: string): Promise<void> {
  await apiFetch<{ status: string }>(`/api/rules/${id}`, { method: 'DELETE' })
}
