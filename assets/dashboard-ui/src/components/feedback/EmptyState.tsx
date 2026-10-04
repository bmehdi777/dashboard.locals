import type { ReactNode } from 'react'

export function EmptyState({
  icon,
  title,
  description,
  action,
}: {
  icon?: ReactNode
  title: string
  description: string
  action?: ReactNode
}) {
  return (
    <div className="state-panel state-panel-empty">
      {icon ? <div className="empty-state-icon">{icon}</div> : null}
      <div>
        <h2>{title}</h2>
        <p>{description}</p>
        {action ? <div className="state-action">{action}</div> : null}
      </div>
    </div>
  )
}
