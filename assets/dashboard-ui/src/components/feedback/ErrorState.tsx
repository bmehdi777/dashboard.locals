import { AlertCircle, RefreshCw } from 'lucide-react'
import { getErrorMessage } from '../../api'
import { Button } from '../ui/Button'

export function ErrorState({
  error,
  onRetry,
  title = 'Impossible de charger ces données',
}: {
  error: unknown
  onRetry?: () => void
  title?: string
}) {
  return (
    <div className="state-panel state-panel-error" role="alert">
      <AlertCircle size={22} aria-hidden="true" />
      <div>
        <h2>{title}</h2>
        <p>{getErrorMessage(error)}</p>
        {onRetry ? (
          <Button variant="outline" size="sm" onClick={onRetry}>
            <RefreshCw size={15} aria-hidden="true" /> Réessayer
          </Button>
        ) : null}
      </div>
    </div>
  )
}
