import { LoaderCircle } from 'lucide-react'

export function LoadingState({ label = 'Chargement en cours…' }: { label?: string }) {
  return (
    <div className="loading-state" role="status" aria-live="polite">
      <LoaderCircle className="spin" size={18} aria-hidden="true" />
      <span>{label}</span>
    </div>
  )
}
