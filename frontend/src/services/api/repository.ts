import type { ConnectedRepo, ListedRepo } from '../../types'
import { ApiRequestError, apiFetch } from './client'

export async function listGitHubRepos(): Promise<ListedRepo[]> {
  const body = await apiFetch<{ repositories: ListedRepo[] }>('/api/github/repositories')
  return body.repositories
}

export async function getConnectedRepo(): Promise<ConnectedRepo | null> {
  try {
    const body = await apiFetch<{ repository: ConnectedRepo }>('/api/repository')
    return body.repository
  } catch (err) {
    if (err instanceof ApiRequestError && (err.status === 404 || err.code === 'REPOSITORY_NOT_CONNECTED')) {
      return null
    }
    throw err
  }
}

export async function connectRepo(githubRepositoryId: number): Promise<ConnectedRepo> {
  const body = await apiFetch<{ repository: ConnectedRepo }>('/api/repository', {
    method: 'POST',
    body: JSON.stringify({ github_repository_id: githubRepositoryId }),
  })
  return body.repository
}

export async function disconnectRepo(): Promise<void> {
  await apiFetch<{ status: string }>('/api/repository', { method: 'DELETE' })
}
