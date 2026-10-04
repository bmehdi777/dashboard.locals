import { useEffect, useRef, useState, type DragEvent } from 'react'
import { Edit3, ExternalLink, GripVertical, Link2, Share2, Trash2 } from 'lucide-react'
import type { Shortcut } from '../../api'
import { Button } from '../../components/ui/Button'
import { formatNumber } from '../../lib/formatters'
import { useRecordShortcutUseMutation } from '../shared/hooks'

export function ShortcutCard({
  shortcut,
  editable = false,
  onEdit,
  onDelete,
  onDragStart,
  onDragOver,
  onDrop,
  onDragEnd,
  isDragging = false,
  isDropTarget = false,
  dragDisabled = false,
}: {
  shortcut: Shortcut
  editable?: boolean
  onEdit?: (shortcut: Shortcut) => void
  onDelete?: (shortcut: Shortcut) => void
  onDragStart?: (shortcut: Shortcut) => void
  onDragOver?: (event: DragEvent<HTMLElement>, shortcut: Shortcut) => void
  onDrop?: (shortcut: Shortcut) => void
  onDragEnd?: () => void
  isDragging?: boolean
  isDropTarget?: boolean
  dragDisabled?: boolean
}) {
  const useMutation = useRecordShortcutUseMutation()
  const [faviconFailed, setFaviconFailed] = useState(false)
  const [shareMessage, setShareMessage] = useState<string | null>(null)
  const [shareError, setShareError] = useState(false)
  const shareToastTimer = useRef<number | null>(null)
  const reorderable = editable && !dragDisabled && Boolean(onDragStart && onDragOver && onDrop)

  useEffect(() => () => {
    if (shareToastTimer.current !== null) window.clearTimeout(shareToastTimer.current)
  }, [])

  function recordUse() {
    useMutation.mutate(shortcut.id)
  }

  async function copyShortcutURL() {
    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(shortcut.url)
      } else {
        const textarea = document.createElement('textarea')
        textarea.value = shortcut.url
        textarea.setAttribute('readonly', '')
        textarea.style.position = 'fixed'
        textarea.style.opacity = '0'
        document.body.appendChild(textarea)
        textarea.select()
        const copied = document.execCommand('copy')
        textarea.remove()
        if (!copied) throw new Error('copy command failed')
      }
      setShareError(false)
      setShareMessage('Lien copié dans le presse-papiers.')
    } catch {
      setShareError(true)
      setShareMessage('Impossible de copier le lien.')
    }
    if (shareToastTimer.current !== null) window.clearTimeout(shareToastTimer.current)
    shareToastTimer.current = window.setTimeout(() => setShareMessage(null), 2400)
  }

  return (
    <article
      className={`shortcut-card${isDragging ? ' shortcut-card-dragging' : ''}${isDropTarget ? ' shortcut-card-drop-target' : ''}`}
      draggable={reorderable}
      onDragStart={(event) => {
        if (!reorderable || !onDragStart) return
        event.dataTransfer.effectAllowed = 'move'
        onDragStart(shortcut)
      }}
      onDragOver={(event) => onDragOver?.(event, shortcut)}
      onDrop={(event) => {
        event.preventDefault()
        onDrop?.(shortcut)
      }}
      onDragEnd={onDragEnd}
    >
      <a
        className="shortcut-card-link"
        href={shortcut.url}
        target="_blank"
        rel="noreferrer"
        onClick={recordUse}
      >
        <div className="shortcut-card-heading">
          <span className="shortcut-card-icon" aria-hidden="true">
            {faviconFailed ? <Link2 size={19} /> : <img src={`/api/v1/shortcuts/${encodeURIComponent(shortcut.id)}/favicon`} alt="" onError={() => setFaviconFailed(true)} />}
          </span>
          <span className="shortcut-card-open"><ExternalLink size={15} aria-hidden="true" /></span>
        </div>
        <strong className="shortcut-card-title">{shortcut.title}</strong>
        {shortcut.description ? <p className="shortcut-card-description">{shortcut.description}</p> : null}
        <span className="shortcut-card-url">{shortcut.url}</span>
      </a>
      <div className="shortcut-card-footer">
        <div className="shortcut-card-footer-info">
          {reorderable ? <GripVertical className="shortcut-card-drag-handle" size={15} aria-hidden="true" /> : null}
          <span>{formatNumber(shortcut.usageCount)} ouverture{shortcut.usageCount === 1 ? '' : 's'}</span>
          {reorderable ? <span className="sr-only">Glisser pour modifier l’ordre</span> : null}
        </div>
        <div className="shortcut-card-actions">
          <Button
            variant="ghost"
            size="icon"
            onClick={(event) => { event.stopPropagation(); void copyShortcutURL() }}
            aria-label={`Partager ${shortcut.title}`}
            title="Copier le lien"
          >
            <Share2 size={15} aria-hidden="true" />
          </Button>
          {editable && onEdit && onDelete ? (
            <Button variant="ghost" size="icon" onClick={() => onEdit(shortcut)} aria-label={`Modifier ${shortcut.title}`}>
              <Edit3 size={15} aria-hidden="true" />
            </Button>
          ) : null}
          {editable && onEdit && onDelete ? (
            <Button variant="ghost" size="icon" onClick={() => onDelete(shortcut)} aria-label={`Supprimer ${shortcut.title}`}>
              <Trash2 size={15} aria-hidden="true" />
            </Button>
          ) : null}
        </div>
      </div>
      {shareMessage ? <div className={`shortcut-share-toast${shareError ? ' shortcut-share-toast-error' : ''}`} role={shareError ? 'alert' : 'status'} aria-live="polite">{shareMessage}</div> : null}
    </article>
  )
}

export function ShortcutGrid({
  shortcuts,
  editable = false,
  onEdit,
  onDelete,
  onDragStart,
  onDragOver,
  onDrop,
  onDragEnd,
  draggedId,
  dropTargetId,
  dragDisabled = false,
  className = '',
}: {
  shortcuts: Shortcut[]
  editable?: boolean
  onEdit?: (shortcut: Shortcut) => void
  onDelete?: (shortcut: Shortcut) => void
  onDragStart?: (shortcut: Shortcut) => void
  onDragOver?: (event: DragEvent<HTMLElement>, shortcut: Shortcut) => void
  onDrop?: (shortcut: Shortcut) => void
  onDragEnd?: () => void
  draggedId?: string | null
  dropTargetId?: string | null
  dragDisabled?: boolean
  className?: string
}) {
  return (
    <div className={`shortcut-grid ${className}`.trim()}>
      {shortcuts.map((shortcut) => (
        <ShortcutCard
          key={`${shortcut.id}:${shortcut.url}`}
          shortcut={shortcut}
          editable={editable}
          onEdit={onEdit}
          onDelete={onDelete}
          onDragStart={onDragStart}
          onDragOver={onDragOver}
          onDrop={onDrop}
          onDragEnd={onDragEnd}
          isDragging={draggedId === shortcut.id}
          isDropTarget={dropTargetId === shortcut.id}
          dragDisabled={dragDisabled}
        />
      ))}
    </div>
  )
}
