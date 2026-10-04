import { useState, type DragEvent, type FormEvent } from 'react'
import { Edit3, Folder, FolderPlus, Link2, Plus, Save, Trash2, X } from 'lucide-react'
import { getErrorMessage } from '../../api'
import type { Shortcut, ShortcutFolder, ShortcutInput } from '../../api'
import { EmptyState } from '../../components/feedback/EmptyState'
import { ErrorState } from '../../components/feedback/ErrorState'
import { LoadingState } from '../../components/feedback/LoadingState'
import { PageHeader } from '../../components/layout/PageHeader'
import { Alert } from '../../components/ui/Alert'
import { Button } from '../../components/ui/Button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../components/ui/Card'
import { Input } from '../../components/ui/Input'
import { Textarea } from '../../components/ui/Textarea'
import {
  useCreateShortcutMutation,
  useCreateShortcutFolderMutation,
  useDeleteShortcutFolderMutation,
  useDeleteShortcutMutation,
  useReorderShortcutsMutation,
  useShortcutFoldersQuery,
  useShortcutsQuery,
  useUpdateShortcutFolderMutation,
  useUpdateShortcutMutation,
} from '../shared/hooks'
import { ShortcutGrid } from './ShortcutCard'

const emptyShortcut: ShortcutInput = {
  title: '',
  url: '',
  description: '',
  folderId: '',
}

export function ShortcutsPage() {
  const shortcutsQuery = useShortcutsQuery('custom')
  const foldersQuery = useShortcutFoldersQuery()

  return (
    <div className="page-stack">
      <PageHeader
        eyebrow="Accès rapides"
        title="Raccourcis"
        description="Rassemblez vos liens importants dans des panneaux accessibles en un clic."
        actions={<Button onClick={() => document.getElementById('shortcut-title')?.focus()}><Plus size={16} aria-hidden="true" /> Ajouter un raccourci</Button>}
      />

      {shortcutsQuery.isLoading || foldersQuery.isLoading ? <LoadingState label="Chargement des raccourcis…" /> : null}
      {shortcutsQuery.isError ? <ErrorState error={shortcutsQuery.error} onRetry={() => void shortcutsQuery.refetch()} title="Les raccourcis sont indisponibles" /> : null}
      {foldersQuery.isError ? <ErrorState error={foldersQuery.error} onRetry={() => void foldersQuery.refetch()} title="Les dossiers sont indisponibles" /> : null}
      {!shortcutsQuery.isLoading && !shortcutsQuery.isError && !foldersQuery.isLoading && !foldersQuery.isError ? (
        <ShortcutWorkspace shortcuts={shortcutsQuery.data ?? []} folders={foldersQuery.data ?? []} />
      ) : null}
    </div>
  )
}

function ShortcutWorkspace({ shortcuts, folders }: { shortcuts: Shortcut[]; folders: ShortcutFolder[] }) {
  const createMutation = useCreateShortcutMutation()
  const updateMutation = useUpdateShortcutMutation()
  const deleteMutation = useDeleteShortcutMutation()
  const reorderMutation = useReorderShortcutsMutation()
  const createFolderMutation = useCreateShortcutFolderMutation()
  const updateFolderMutation = useUpdateShortcutFolderMutation()
  const deleteFolderMutation = useDeleteShortcutFolderMutation()
  const [form, setForm] = useState<ShortcutInput>(emptyShortcut)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [message, setMessage] = useState<string | null>(null)
  const [folderName, setFolderName] = useState('')
  const [editingFolderId, setEditingFolderId] = useState<string | null>(null)
  const [folderMessage, setFolderMessage] = useState<string | null>(null)
  const [folderError, setFolderError] = useState(false)
  const [orderMessage, setOrderMessage] = useState<string | null>(null)
  const [orderError, setOrderError] = useState(false)
  const shortcutSourceKey = JSON.stringify(shortcuts.map((shortcut) => [
    shortcut.id,
    shortcut.position,
    shortcut.updatedAt,
    shortcut.title,
    shortcut.url,
    shortcut.description,
    shortcut.folderId,
    shortcut.usageCount,
    shortcut.lastUsedAt,
  ]))
  const [orderState, setOrderState] = useState({ sourceKey: shortcutSourceKey, shortcuts })
  const [draggedId, setDraggedId] = useState<string | null>(null)
  const [dropTargetId, setDropTargetId] = useState<string | null>(null)
  const [dropFolderKey, setDropFolderKey] = useState<string | null>(null)
  const orderedShortcuts = orderState.sourceKey === shortcutSourceKey ? orderState.shortcuts : shortcuts

  function editShortcut(shortcut: Shortcut) {
    setEditingId(shortcut.id)
    setForm({ title: shortcut.title, url: shortcut.url, description: shortcut.description, folderId: shortcut.folderId ?? '' })
    setMessage(null)
    document.getElementById('shortcut-title')?.focus()
  }

  function cancelEdit() {
    setEditingId(null)
    setForm(emptyShortcut)
  }

  function submitShortcut(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setMessage(null)
    const shortcut = {
      title: form.title.trim(),
      url: form.url.trim(),
      description: form.description.trim(),
      folderId: form.folderId ?? '',
    }
    if (!shortcut.title || !shortcut.url) {
      setMessage('Le titre et l’URL sont obligatoires.')
      return
    }

    if (editingId) {
      updateMutation.mutate(
        { id: editingId, shortcut },
        {
          onSuccess: () => {
            setMessage('Le raccourci a été mis à jour.')
            cancelEdit()
          },
          onError: (error) => setMessage(getErrorMessage(error)),
        },
      )
      return
    }

    createMutation.mutate(shortcut, {
      onSuccess: () => {
        setMessage('Le raccourci a été ajouté.')
        setForm(emptyShortcut)
      },
      onError: (error) => setMessage(getErrorMessage(error)),
    })
  }

  function removeShortcut(shortcut: Shortcut) {
    if (!window.confirm(`Supprimer le raccourci « ${shortcut.title} » ?`)) return
    setMessage(null)
    deleteMutation.mutate(shortcut.id, {
      onSuccess: () => {
        setMessage('Le raccourci a été supprimé.')
        if (editingId === shortcut.id) cancelEdit()
      },
      onError: (error) => setMessage(getErrorMessage(error)),
    })
  }

  function submitFolder(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setFolderMessage(null)
    setFolderError(false)
    const name = folderName.trim()
    if (!name) {
      setFolderError(true)
      setFolderMessage('Le nom du dossier est obligatoire.')
      return
    }
    if (editingFolderId) {
      updateFolderMutation.mutate(
        { id: editingFolderId, folder: { name } },
        {
          onSuccess: () => {
            setFolderMessage('Le dossier a été renommé.')
            setEditingFolderId(null)
            setFolderName('')
          },
          onError: (error) => { setFolderError(true); setFolderMessage(getErrorMessage(error)) },
        },
      )
      return
    }
    createFolderMutation.mutate(
      { name },
      {
        onSuccess: () => {
          setFolderMessage('Le dossier a été ajouté.')
          setFolderName('')
        },
        onError: (error) => { setFolderError(true); setFolderMessage(getErrorMessage(error)) },
      },
    )
  }

  function editFolder(folder: ShortcutFolder) {
    setEditingFolderId(folder.id)
    setFolderName(folder.name)
    setFolderMessage(null)
    setFolderError(false)
  }

  function cancelFolderEdit() {
    setEditingFolderId(null)
    setFolderName('')
  }

  function removeFolder(folder: ShortcutFolder) {
    if (!window.confirm(`Supprimer le dossier « ${folder.name} » ? Les raccourcis seront déplacés dans « Sans dossier ».`)) return
    setFolderMessage(null)
    setFolderError(false)
    deleteFolderMutation.mutate(folder.id, {
      onSuccess: () => setFolderMessage('Le dossier a été supprimé.'),
      onError: (error) => { setFolderError(true); setFolderMessage(getErrorMessage(error)) },
    })
  }

  function handleDragStart(shortcut: Shortcut) {
    if (reorderMutation.isPending) return
    setDraggedId(shortcut.id)
    setDropTargetId(null)
    setDropFolderKey(null)
    setOrderMessage(null)
  }

  function handleDragOver(event: DragEvent<HTMLElement>, shortcut: Shortcut) {
    event.preventDefault()
    event.stopPropagation()
    if (!draggedId || draggedId === shortcut.id) return
    setDropTargetId((current) => current === shortcut.id ? current : shortcut.id)
    setDropFolderKey(null)
  }

  function handleDrop(shortcut: Shortcut) {
    if (!draggedId || draggedId === shortcut.id) {
      resetDragState()
      return
    }
    const nextOrder = moveShortcut(orderedShortcuts, draggedId, shortcut.id)
    commitOrder(nextOrder, shortcut.folderId)
  }

  function handleFolderDragOver(event: DragEvent<HTMLElement>, folderId: string | null) {
    event.preventDefault()
    if (!draggedId) return
    setDropTargetId(null)
    setDropFolderKey(folderId ?? 'uncategorized')
  }

  function handleFolderDrop(folderId: string | null) {
    if (!draggedId) {
      resetDragState()
      return
    }
    commitOrder(moveShortcutToFolder(orderedShortcuts, draggedId, folderId), folderId)
  }

  function commitOrder(nextOrder: Shortcut[], folderId: string | null | undefined) {
    if (!draggedId) return
    const updatedOrder = nextOrder.map((item) => item.id === draggedId
      ? { ...item, folderId: folderId ?? undefined }
      : item)
    setOrderState({ sourceKey: shortcutSourceKey, shortcuts: updatedOrder })
    resetDragState()
    setOrderMessage(null)
    setOrderError(false)
    reorderMutation.mutate(updatedOrder.map((item) => ({ id: item.id, folderId: item.folderId })), {
      onSuccess: () => setOrderMessage('L’ordre des raccourcis a été enregistré.'),
      onError: (error) => {
        setOrderState({ sourceKey: shortcutSourceKey, shortcuts })
        setOrderError(true)
        setOrderMessage(getErrorMessage(error))
      },
    })
  }

  function resetDragState() {
    setDraggedId(null)
    setDropTargetId(null)
    setDropFolderKey(null)
  }

  const formPending = createMutation.isPending || updateMutation.isPending
  const folderPending = createFolderMutation.isPending || updateFolderMutation.isPending || deleteFolderMutation.isPending
  const isValidationMessage = message?.includes('obligatoires') ?? false
  const folderGroups = buildFolderGroups(orderedShortcuts, folders)

  return (
    <>
      <Card className="shortcut-folders-card">
        <CardHeader>
          <CardTitle>Dossiers</CardTitle>
          <CardDescription>Créez des dossiers pour organiser vos raccourcis. Un lien peut être déplacé d’un dossier à un autre par glisser-déposer.</CardDescription>
        </CardHeader>
        <CardContent>
          {folderMessage ? <Alert variant={folderError ? 'danger' : 'success'}>{folderMessage}</Alert> : null}
          {folders.length ? (
            <div className="shortcut-folder-list">
              {folders.map((folder) => (
                <div className="shortcut-folder-row" key={folder.id}>
                  <span className="shortcut-folder-row-icon"><Folder size={16} aria-hidden="true" /></span>
                  <strong>{folder.name}</strong>
                  <span className="muted-text">{orderedShortcuts.filter((shortcut) => shortcut.folderId === folder.id).length} raccourci(s)</span>
                  <div className="shortcut-folder-row-actions">
                    <Button variant="ghost" size="icon" onClick={() => editFolder(folder)} aria-label={`Renommer ${folder.name}`}>
                      <Edit3 size={15} aria-hidden="true" />
                    </Button>
                    <Button variant="ghost" size="icon" onClick={() => removeFolder(folder)} disabled={folderPending} aria-label={`Supprimer ${folder.name}`}>
                      <Trash2 size={15} aria-hidden="true" />
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          ) : null}
          <form className="shortcut-folder-form" onSubmit={submitFolder}>
            <div className="form-field">
              <label htmlFor="shortcut-folder-name">{editingFolderId ? 'Nom du dossier' : 'Nouveau dossier'}</label>
              <Input id="shortcut-folder-name" value={folderName} onChange={(event) => setFolderName(event.target.value)} placeholder="Documentation" maxLength={100} />
            </div>
            <div className="form-actions">
              {editingFolderId ? <Button type="button" variant="ghost" onClick={cancelFolderEdit}><X size={16} aria-hidden="true" /> Annuler</Button> : null}
              <Button type="submit" disabled={folderPending}>
                {editingFolderId ? <Save size={16} aria-hidden="true" /> : <FolderPlus size={16} aria-hidden="true" />}
                {folderPending ? 'Enregistrement…' : editingFolderId ? 'Renommer' : 'Ajouter le dossier'}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      <Card className="shortcut-panels-card">
        <CardHeader>
          <CardTitle>Vos raccourcis</CardTitle>
          <CardDescription>Ouvrez un lien pour l’utiliser ou glissez les raccourcis pour modifier leur ordre. Le serveur conserve le nombre d’ouvertures afin d’afficher les plus utilisés sur l’accueil.</CardDescription>
        </CardHeader>
        <CardContent>
          {orderMessage ? <Alert variant={orderError ? 'danger' : 'success'}>{orderMessage}</Alert> : null}
          {orderedShortcuts.length ? (
            <div className="shortcut-folder-groups">
              {folderGroups.map((group) => {
                const folderKey = group.id ?? 'uncategorized'
                const isDropTarget = dropFolderKey === folderKey
                return (
                  <section className="shortcut-folder-section" key={folderKey}>
                    <div
                      className={`shortcut-folder-heading${isDropTarget ? ' shortcut-folder-heading-drop-target' : ''}`}
                      onDragOver={(event) => handleFolderDragOver(event, group.id)}
                      onDrop={(event) => { event.preventDefault(); handleFolderDrop(group.id) }}
                    >
                      <div className="shortcut-folder-heading-title">
                        <Folder size={17} aria-hidden="true" />
                        <h3>{group.name}</h3>
                        <span>{group.shortcuts.length}</span>
                      </div>
                      <span className="shortcut-folder-heading-hint">Déposer ici pour déplacer</span>
                    </div>
                    {group.shortcuts.length ? (
                      <ShortcutGrid
                        shortcuts={group.shortcuts}
                        editable
                        onEdit={editShortcut}
                        onDelete={removeShortcut}
                        onDragStart={handleDragStart}
                        onDragOver={handleDragOver}
                        onDrop={handleDrop}
                        onDragEnd={resetDragState}
                        draggedId={draggedId}
                        dropTargetId={dropTargetId}
                        dragDisabled={reorderMutation.isPending}
                        className="shortcut-grid-manager"
                      />
                    ) : (
                      <div
                        className={`shortcut-folder-empty${isDropTarget ? ' shortcut-folder-empty-drop-target' : ''}`}
                        onDragOver={(event) => handleFolderDragOver(event, group.id)}
                        onDrop={(event) => { event.preventDefault(); handleFolderDrop(group.id) }}
                      >
                        Déposez un raccourci dans ce dossier.
                      </div>
                    )}
                  </section>
                )
              })}
            </div>
          ) : (
            <EmptyState icon={<Link2 size={23} aria-hidden="true" />} title="Aucun raccourci" description="Ajoutez votre premier lien pour composer votre espace d’accès rapides." />
          )}
        </CardContent>
      </Card>

      <Card className="shortcut-form-card">
        <CardHeader>
          <CardTitle>{editingId ? 'Modifier un raccourci' : 'Ajouter un raccourci'}</CardTitle>
          <CardDescription>Seuls les liens HTTP et HTTPS sont acceptés.</CardDescription>
        </CardHeader>
        <CardContent>
          {message ? <Alert variant={isValidationMessage ? 'danger' : 'success'}>{message}</Alert> : null}
          <form className="shortcut-form" onSubmit={submitShortcut}>
              <div className="form-grid form-grid-two">
              <div className="form-field">
                <label htmlFor="shortcut-title">Titre</label>
                <Input id="shortcut-title" value={form.title} onChange={(event) => setForm((current) => ({ ...current, title: event.target.value }))} placeholder="Documentation" maxLength={200} />
              </div>
                <div className="form-field">
                  <label htmlFor="shortcut-url">URL</label>
                  <Input id="shortcut-url" type="url" value={form.url} onChange={(event) => setForm((current) => ({ ...current, url: event.target.value }))} placeholder="https://exemple.com" maxLength={2048} />
                </div>
                <div className="form-field">
                  <label htmlFor="shortcut-folder">Dossier</label>
                  <select className="select" id="shortcut-folder" value={form.folderId ?? ''} onChange={(event) => setForm((current) => ({ ...current, folderId: event.target.value }))}>
                    <option value="">Sans dossier</option>
                    {folders.map((folder) => <option value={folder.id} key={folder.id}>{folder.name}</option>)}
                  </select>
                </div>
              </div>
            <div className="form-field">
              <label htmlFor="shortcut-description">Description <span className="muted-text">(facultatif)</span></label>
              <Textarea id="shortcut-description" value={form.description} onChange={(event) => setForm((current) => ({ ...current, description: event.target.value }))} placeholder="Une courte description de ce lien" maxLength={500} rows={3} />
            </div>
            <div className="form-actions">
              {editingId ? <Button type="button" variant="ghost" onClick={cancelEdit}><X size={16} aria-hidden="true" /> Annuler</Button> : null}
              <Button type="submit" disabled={formPending || deleteMutation.isPending}>
                {editingId ? <Save size={16} aria-hidden="true" /> : <Plus size={16} aria-hidden="true" />}
                {formPending ? 'Enregistrement…' : editingId ? 'Enregistrer' : 'Ajouter le raccourci'}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </>
  )
}

function moveShortcut(shortcuts: Shortcut[], sourceId: string, targetId: string): Shortcut[] {
  const sourceIndex = shortcuts.findIndex((shortcut) => shortcut.id === sourceId)
  if (sourceIndex < 0) return shortcuts

  const nextOrder = [...shortcuts]
  const [movedShortcut] = nextOrder.splice(sourceIndex, 1)
  if (!movedShortcut) return shortcuts

  const targetIndex = nextOrder.findIndex((shortcut) => shortcut.id === targetId)
  if (targetIndex < 0) return shortcuts

  nextOrder.splice(targetIndex, 0, movedShortcut)
  return nextOrder
}

function moveShortcutToFolder(shortcuts: Shortcut[], sourceId: string, folderId: string | null): Shortcut[] {
  const sourceIndex = shortcuts.findIndex((shortcut) => shortcut.id === sourceId)
  if (sourceIndex < 0) return shortcuts

  const nextOrder = [...shortcuts]
  const [movedShortcut] = nextOrder.splice(sourceIndex, 1)
  if (!movedShortcut) return shortcuts

  const lastTargetIndex = nextOrder.reduce(
    (lastIndex, shortcut, index) => (shortcut.folderId === (folderId ?? undefined) ? index : lastIndex),
    -1,
  )
  nextOrder.splice(lastTargetIndex + 1, 0, movedShortcut)
  return nextOrder
}

function buildFolderGroups(shortcuts: Shortcut[], folders: ShortcutFolder[]) {
  return [
    ...folders.map((folder) => ({
      id: folder.id,
      name: folder.name,
      shortcuts: shortcuts.filter((shortcut) => shortcut.folderId === folder.id),
    })),
    {
      id: null,
      name: 'Sans dossier',
      shortcuts: shortcuts.filter((shortcut) => !shortcut.folderId),
    },
  ]
}
