type StatusBadgeProps = {
  status: string
}

const LABELS: Record<string, string> = {
  processed: 'Processed',
  pending: 'Pending',
  processing: 'Processing',
  completed: 'Completed',
  failed: 'Failed',
  enabled: 'Enabled',
  disabled: 'Disabled',
}

export function StatusBadge({ status }: StatusBadgeProps) {
  const key = status.toLowerCase()
  const label = LABELS[key] ?? status
  const tone =
    key === 'completed' || key === 'processed' || key === 'enabled'
      ? 'ok'
      : key === 'failed'
        ? 'bad'
        : key === 'processing' || key === 'pending'
          ? 'warn'
          : 'neutral'
  const prefix =
    key === 'completed' || key === 'processed' || key === 'enabled'
      ? '✓ '
      : key === 'failed'
        ? '⚠ '
        : key === 'processing' || key === 'pending'
          ? '● '
          : ''

  return (
    <span className={`badge badge-${tone}`} title={label}>
      {prefix}
      {label}
    </span>
  )
}
