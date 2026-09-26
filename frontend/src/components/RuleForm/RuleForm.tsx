import { FormEvent, useEffect, useState } from 'react'
import type { Rule, RuleInput } from '../../types'

type RuleFormProps = {
  initial?: Rule | null
  busy: boolean
  error: string | null
  onSubmit: (input: RuleInput) => Promise<void>
  onCancel: () => void
}

function configFromRule(rule?: Rule | null): {
  label: string
  comment: string
  message: string
  use_ai: boolean
  append_ai_summary: boolean
} {
  const cfg = rule?.action_config ?? {}
  return {
    label: typeof cfg.label === 'string' ? cfg.label : '',
    comment: typeof cfg.comment === 'string' ? cfg.comment : '',
    message: typeof cfg.message === 'string' ? cfg.message : '',
    use_ai: cfg.use_ai === true,
    append_ai_summary: cfg.append_ai_summary === true,
  }
}

export function RuleForm({ initial, busy, error, onSubmit, onCancel }: RuleFormProps) {
  const [name, setName] = useState(initial?.name ?? '')
  const [enabled, setEnabled] = useState(initial?.enabled ?? true)
  const [eventType, setEventType] = useState(initial?.event_type ?? 'issues')
  const [keyword, setKeyword] = useState(initial?.keyword ?? '')
  const [author, setAuthor] = useState(initial?.author ?? '')
  const [labels, setLabels] = useState((initial?.required_labels ?? []).join(', '))
  const [actionType, setActionType] = useState(initial?.action_type ?? 'github_label')
  const [cfg, setCfg] = useState(configFromRule(initial))
  const [localError, setLocalError] = useState<string | null>(null)

  useEffect(() => {
    setName(initial?.name ?? '')
    setEnabled(initial?.enabled ?? true)
    setEventType(initial?.event_type ?? 'issues')
    setKeyword(initial?.keyword ?? '')
    setAuthor(initial?.author ?? '')
    setLabels((initial?.required_labels ?? []).join(', '))
    setActionType(initial?.action_type ?? 'github_label')
    setCfg(configFromRule(initial))
  }, [initial])

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setLocalError(null)
    if (!name.trim()) {
      setLocalError('Rule name is required.')
      return
    }
    if (!eventType) {
      setLocalError('Please select an event type.')
      return
    }
    let action_config: Record<string, unknown> = {}
    if (actionType === 'github_label') {
      if (!cfg.label.trim() && !cfg.use_ai) {
        setLocalError('Label name is required (or enable AI label suggestion).')
        return
      }
      action_config = { label: cfg.label.trim(), ...(cfg.use_ai ? { use_ai: true } : {}) }
    } else if (actionType === 'github_comment') {
      if (!cfg.comment.trim() && !cfg.use_ai && !cfg.append_ai_summary) {
        setLocalError('Comment is required (or enable AI summary options).')
        return
      }
      action_config = {
        comment: cfg.comment.trim(),
        ...(cfg.use_ai ? { use_ai: true } : {}),
        ...(cfg.append_ai_summary ? { append_ai_summary: true } : {}),
      }
    } else if (actionType === 'slack_notification') {
      if (!cfg.message.trim()) {
        setLocalError('Message is required.')
        return
      }
      action_config = {
        message: cfg.message.trim(),
        ...(cfg.append_ai_summary ? { append_ai_summary: true } : {}),
      }
    }

    const required_labels = labels
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean)

    const input: RuleInput = {
      name: name.trim(),
      enabled,
      event_type: eventType,
      keyword: keyword.trim() || null,
      author: author.trim() || null,
      required_labels,
      action_type: actionType,
      action_config,
    }
    await onSubmit(input)
  }

  return (
    <form className="rule-form" onSubmit={(e) => void handleSubmit(e)}>
      <h3>{initial ? 'Edit rule' : 'Create rule'}</h3>
      {(localError || error) && (
        <p className="error" role="alert">
          {localError ?? error}
        </p>
      )}

      <label>
        Rule name
        <input value={name} onChange={(e) => setName(e.target.value)} required maxLength={120} />
      </label>

      <label>
        Event type
        <select value={eventType} onChange={(e) => setEventType(e.target.value)}>
          <option value="issues">Issue</option>
          <option value="pull_request">Pull request</option>
        </select>
      </label>

      <label>
        Keyword <span className="optional">(optional)</span>
        <input value={keyword} onChange={(e) => setKeyword(e.target.value)} placeholder="e.g. bug" />
      </label>

      <label>
        Author <span className="optional">(optional)</span>
        <input value={author} onChange={(e) => setAuthor(e.target.value)} placeholder="GitHub login" />
      </label>

      <label>
        Required labels <span className="optional">(comma-separated, optional)</span>
        <input value={labels} onChange={(e) => setLabels(e.target.value)} placeholder="bug, triage" />
      </label>

      <label>
        Action
        <select value={actionType} onChange={(e) => setActionType(e.target.value)}>
          <option value="github_label">GitHub label</option>
          <option value="github_comment">GitHub comment</option>
          <option value="slack_notification">Slack notification</option>
        </select>
      </label>

      {actionType === 'github_label' && (
        <label>
          Label name
          <input value={cfg.label} onChange={(e) => setCfg({ ...cfg, label: e.target.value })} />
        </label>
      )}
      {actionType === 'github_comment' && (
        <label>
          Comment
          <textarea
            rows={3}
            value={cfg.comment}
            onChange={(e) => setCfg({ ...cfg, comment: e.target.value })}
          />
        </label>
      )}
      {actionType === 'slack_notification' && (
        <label>
          Message
          <textarea
            rows={3}
            value={cfg.message}
            onChange={(e) => setCfg({ ...cfg, message: e.target.value })}
          />
          <span className="field-hint">Slack destination is configured on the server, not in this form.</span>
        </label>
      )}

      <label className="check">
        <input
          type="checkbox"
          checked={cfg.use_ai}
          onChange={(e) => setCfg({ ...cfg, use_ai: e.target.checked })}
        />
        Optional AI assist (label/comment from model when AI_ENABLED on server)
      </label>
      <label className="check">
        <input
          type="checkbox"
          checked={cfg.append_ai_summary}
          onChange={(e) => setCfg({ ...cfg, append_ai_summary: e.target.checked })}
        />
        Append AI summary (comment/Slack) when available
      </label>
      <p className="field-hint">
        AI is optional. If the provider is down or disabled, rules still run with your static label/comment/message.
      </p>

      <label className="check">
        <input type="checkbox" checked={enabled} onChange={(e) => setEnabled(e.target.checked)} />
        Enabled
      </label>

      <div className="form-actions">
        <button type="button" className="btn ghost" onClick={onCancel} disabled={busy}>
          Cancel
        </button>
        <button type="submit" className="btn" disabled={busy}>
          {busy ? 'Saving…' : initial ? 'Save changes' : 'Create rule'}
        </button>
      </div>
    </form>
  )
}
