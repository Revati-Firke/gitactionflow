import { useCallback, useEffect, useState } from 'react'
import { ActionList } from '../components/ActionList/ActionList'
import { EventList } from '../components/EventList/EventList'
import { RepositoryCard } from '../components/RepositoryCard/RepositoryCard'
import { RuleList } from '../components/RuleList/RuleList'
import {
  ApiRequestError,
  connectRepo,
  createRule,
  deleteRule,
  disconnectRepo,
  getConnectedRepo,
  listActions,
  listEvents,
  listGitHubRepos,
  listRules,
  updateRule,
} from '../services/api'
import type {
  ActionSummary,
  ConnectedRepo,
  ListedRepo,
  Rule,
  RuleInput,
  User,
  WebhookEventSummary,
} from '../types'

type DashboardPageProps = {
  user: User
}

export function DashboardPage(_props: DashboardPageProps) {
  const [connected, setConnected] = useState<ConnectedRepo | null>(null)
  const [candidates, setCandidates] = useState<ListedRepo[]>([])
  const [picking, setPicking] = useState(false)
  const [repoLoading, setRepoLoading] = useState(true)
  const [repoBusy, setRepoBusy] = useState(false)
  const [repoError, setRepoError] = useState<string | null>(null)

  const [rules, setRules] = useState<Rule[]>([])
  const [rulesLoading, setRulesLoading] = useState(false)
  const [rulesError, setRulesError] = useState<string | null>(null)
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<Rule | null>(null)
  const [formBusy, setFormBusy] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

  const [events, setEvents] = useState<WebhookEventSummary[]>([])
  const [eventsLoading, setEventsLoading] = useState(false)
  const [eventsError, setEventsError] = useState<string | null>(null)
  const [selectedEvent, setSelectedEvent] = useState<string | null>(null)

  const [actions, setActions] = useState<ActionSummary[]>([])
  const [actionsLoading, setActionsLoading] = useState(false)
  const [actionsError, setActionsError] = useState<string | null>(null)

  const refreshRules = useCallback(async (hasRepo: boolean) => {
    if (!hasRepo) {
      setRules([])
      setRulesError(null)
      return
    }
    setRulesLoading(true)
    setRulesError(null)
    try {
      setRules(await listRules())
    } catch (err) {
      setRulesError(err instanceof Error ? err.message : 'failed to load rules')
    } finally {
      setRulesLoading(false)
    }
  }, [])

  const refreshEvents = useCallback(async (hasRepo: boolean) => {
    if (!hasRepo) {
      setEvents([])
      setEventsError(null)
      return
    }
    setEventsLoading(true)
    setEventsError(null)
    try {
      setEvents(await listEvents(1, 20))
    } catch (err) {
      if (err instanceof ApiRequestError && err.status === 404) {
        setEvents([])
      } else {
        setEventsError(err instanceof Error ? err.message : 'failed to load events')
      }
    } finally {
      setEventsLoading(false)
    }
  }, [])

  const refreshActions = useCallback(async (hasRepo: boolean) => {
    if (!hasRepo) {
      setActions([])
      setActionsError(null)
      return
    }
    setActionsLoading(true)
    setActionsError(null)
    try {
      setActions(await listActions(1, 20))
    } catch (err) {
      if (err instanceof ApiRequestError && err.status === 404) {
        setActions([])
      } else {
        setActionsError(err instanceof Error ? err.message : 'failed to load actions')
      }
    } finally {
      setActionsLoading(false)
    }
  }, [])

  const refreshAll = useCallback(async () => {
    setRepoLoading(true)
    setRepoError(null)
    try {
      const repo = await getConnectedRepo()
      setConnected(repo)
      setPicking(false)
      await Promise.all([refreshRules(!!repo), refreshEvents(!!repo), refreshActions(!!repo)])
    } catch (err) {
      setRepoError(err instanceof Error ? err.message : 'failed to load repository')
    } finally {
      setRepoLoading(false)
    }
  }, [refreshActions, refreshEvents, refreshRules])

  useEffect(() => {
    void refreshAll()
  }, [refreshAll])

  async function onStartPick() {
    setRepoBusy(true)
    setRepoError(null)
    try {
      setCandidates(await listGitHubRepos())
      setPicking(true)
    } catch (err) {
      setRepoError(err instanceof Error ? err.message : 'failed to list repositories')
    } finally {
      setRepoBusy(false)
    }
  }

  async function onConnect(id: number) {
    setRepoBusy(true)
    setRepoError(null)
    try {
      const repo = await connectRepo(id)
      setConnected(repo)
      setPicking(false)
      setCandidates([])
      await Promise.all([refreshRules(true), refreshEvents(true), refreshActions(true)])
    } catch (err) {
      setRepoError(err instanceof Error ? err.message : 'connect failed')
    } finally {
      setRepoBusy(false)
    }
  }

  async function onDisconnect() {
    setRepoBusy(true)
    setRepoError(null)
    try {
      await disconnectRepo()
      setConnected(null)
      setFormOpen(false)
      setEditing(null)
      setRules([])
      setEvents([])
      setActions([])
    } catch (err) {
      setRepoError(err instanceof Error ? err.message : 'disconnect failed')
    } finally {
      setRepoBusy(false)
    }
  }

  async function onSubmitRule(input: RuleInput) {
    setFormBusy(true)
    setFormError(null)
    try {
      if (editing) {
        await updateRule(editing.id, input)
      } else {
        await createRule(input)
      }
      setFormOpen(false)
      setEditing(null)
      await refreshRules(true)
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'failed to save rule')
    } finally {
      setFormBusy(false)
    }
  }

  async function onToggle(rule: Rule) {
    try {
      await updateRule(rule.id, {
        name: rule.name,
        enabled: !rule.enabled,
        event_type: rule.event_type,
        keyword: rule.keyword,
        author: rule.author,
        required_labels: rule.required_labels ?? [],
        action_type: rule.action_type,
        action_config: rule.action_config ?? {},
      })
      await refreshRules(true)
    } catch (err) {
      setRulesError(err instanceof Error ? err.message : 'failed to update rule')
    }
  }

  async function onDelete(rule: Rule) {
    try {
      await deleteRule(rule.id)
      if (editing?.id === rule.id) {
        setFormOpen(false)
        setEditing(null)
      }
      await refreshRules(true)
    } catch (err) {
      setRulesError(err instanceof Error ? err.message : 'failed to delete rule')
    }
  }

  return (
    <div className="dashboard">
      <p className="page-intro muted">Monitor automation for your connected repository.</p>

      <RepositoryCard
        connected={connected}
        candidates={candidates}
        loading={repoLoading}
        connecting={repoBusy}
        picking={picking}
        onStartPick={() => void onStartPick()}
        onCancelPick={() => setPicking(false)}
        onConnect={(id) => void onConnect(id)}
        onDisconnect={() => void onDisconnect()}
        error={repoError}
      />

      <RuleList
        rules={rules}
        loading={rulesLoading}
        error={rulesError}
        formOpen={formOpen}
        editing={editing}
        formBusy={formBusy}
        formError={formError}
        repoConnected={!!connected}
        onOpenCreate={() => {
          setEditing(null)
          setFormError(null)
          setFormOpen(true)
        }}
        onEdit={(rule) => {
          setEditing(rule)
          setFormError(null)
          setFormOpen(true)
        }}
        onCancelForm={() => {
          setFormOpen(false)
          setEditing(null)
          setFormError(null)
        }}
        onSubmit={onSubmitRule}
        onToggle={(r) => void onToggle(r)}
        onDelete={(r) => void onDelete(r)}
        onRetry={() => void refreshRules(!!connected)}
      />

      <EventList
        events={events}
        loading={eventsLoading}
        error={eventsError}
        repoConnected={!!connected}
        selectedId={selectedEvent}
        onSelect={setSelectedEvent}
        onRefresh={() => void refreshEvents(!!connected)}
      />

      <ActionList
        actions={actions}
        loading={actionsLoading}
        error={actionsError}
        repoConnected={!!connected}
        onRefresh={() => void refreshActions(!!connected)}
      />
    </div>
  )
}
