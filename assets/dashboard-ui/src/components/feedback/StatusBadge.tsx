import { Badge } from '../ui/Badge'
import type { HealthState, OpenCodeState } from '../../api'

export function HealthBadge({ status }: { status: HealthState }) {
  const variant =
    status === 'ok' ? 'success' : status === 'degraded' ? 'warning' : 'danger'
  const label =
    status === 'ok' ? 'Opérationnel' : status === 'degraded' ? 'Dégradé' : 'Indisponible'
  return <Badge variant={variant}>{label}</Badge>
}

export function OpenCodeBadge({ state }: { state: OpenCodeState }) {
  const labels: Record<OpenCodeState, string> = {
    available: 'OpenCode connecté',
    missing: 'OpenCode absent',
    stopped: 'OpenCode arrêté',
    incompatible: 'OpenCode incompatible',
    unauthenticated: 'OpenCode non authentifié',
    unknown: 'OpenCode non détecté',
  }
  const variant = state === 'available' ? 'success' : state === 'unknown' ? 'neutral' : 'warning'
  return <Badge variant={variant}>{labels[state]}</Badge>
}
