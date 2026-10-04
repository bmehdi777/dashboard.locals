import type { ReactNode } from 'react'
import { Card, CardContent } from '../ui/Card'

export function MetricCard({
  label,
  value,
  detail,
  icon,
  tone = 'violet',
}: {
  label: string
  value: string
  detail?: string
  icon: ReactNode
  tone?: 'violet' | 'blue' | 'green' | 'orange'
}) {
  return (
    <Card className="metric-card">
      <CardContent>
        <div className={`metric-icon metric-icon-${tone}`}>{icon}</div>
        <p className="metric-label">{label}</p>
        <strong className="metric-value">{value}</strong>
        {detail ? <p className="metric-detail">{detail}</p> : null}
      </CardContent>
    </Card>
  )
}
