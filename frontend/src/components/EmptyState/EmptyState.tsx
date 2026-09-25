import type { ReactNode } from 'react'

type EmptyStateProps = {
  title: string
  description: string
  action?: ReactNode
}

export function EmptyState({ title, description, action }: EmptyStateProps) {
  return (
    <div className="empty">
      <p className="empty-title">{title}</p>
      <p className="muted">{description}</p>
      {action}
    </div>
  )
}
