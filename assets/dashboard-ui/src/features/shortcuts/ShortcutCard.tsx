import { Edit3, ExternalLink, Link2, Trash2 } from 'lucide-react'
import type { Shortcut } from '../../api'
import { Button } from '../../components/ui/Button'
import { formatNumber } from '../../lib/formatters'
import { useRecordShortcutUseMutation } from '../shared/hooks'

export function ShortcutCard({
  shortcut,
  editable = false,
  onEdit,
  onDelete,
}: {
  shortcut: Shortcut
  editable?: boolean
  onEdit?: (shortcut: Shortcut) => void
  onDelete?: (shortcut: Shortcut) => void
}) {
  const useMutation = useRecordShortcutUseMutation()

  function recordUse() {
    useMutation.mutate(shortcut.id)
  }

  return (
    <article className="shortcut-card">
      <a
        className="shortcut-card-link"
        href={shortcut.url}
        target="_blank"
        rel="noreferrer"
        onClick={recordUse}
      >
        <div className="shortcut-card-heading">
          <span className="shortcut-card-icon" aria-hidden="true"><Link2 size={19} /></span>
          <span className="shortcut-card-open"><ExternalLink size={15} aria-hidden="true" /></span>
        </div>
        <strong className="shortcut-card-title">{shortcut.title}</strong>
        {shortcut.description ? <p className="shortcut-card-description">{shortcut.description}</p> : null}
        <span className="shortcut-card-url">{shortcut.url}</span>
      </a>
      <div className="shortcut-card-footer">
        <span>{formatNumber(shortcut.usageCount)} ouverture{shortcut.usageCount === 1 ? '' : 's'}</span>
        {editable && onEdit && onDelete ? (
          <div className="shortcut-card-actions">
            <Button variant="ghost" size="icon" onClick={() => onEdit(shortcut)} aria-label={`Modifier ${shortcut.title}`}>
              <Edit3 size={15} aria-hidden="true" />
            </Button>
            <Button variant="ghost" size="icon" onClick={() => onDelete(shortcut)} aria-label={`Supprimer ${shortcut.title}`}>
              <Trash2 size={15} aria-hidden="true" />
            </Button>
          </div>
        ) : null}
      </div>
    </article>
  )
}

export function ShortcutGrid({
  shortcuts,
  editable = false,
  onEdit,
  onDelete,
  className = '',
}: {
  shortcuts: Shortcut[]
  editable?: boolean
  onEdit?: (shortcut: Shortcut) => void
  onDelete?: (shortcut: Shortcut) => void
  className?: string
}) {
  return (
    <div className={`shortcut-grid ${className}`.trim()}>
      {shortcuts.map((shortcut) => (
        <ShortcutCard
          key={shortcut.id}
          shortcut={shortcut}
          editable={editable}
          onEdit={onEdit}
          onDelete={onDelete}
        />
      ))}
    </div>
  )
}
