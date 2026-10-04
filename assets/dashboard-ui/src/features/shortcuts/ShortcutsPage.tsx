import { useState, type FormEvent } from 'react'
import { Link2, Plus, Save, X } from 'lucide-react'
import { getErrorMessage } from '../../api'
import type { Shortcut, ShortcutInput } from '../../api'
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
  useDeleteShortcutMutation,
  useShortcutsQuery,
  useUpdateShortcutMutation,
} from '../shared/hooks'
import { ShortcutGrid } from './ShortcutCard'

const emptyShortcut: ShortcutInput = {
  title: '',
  url: '',
  description: '',
}

export function ShortcutsPage() {
  const shortcutsQuery = useShortcutsQuery('recent')

  return (
    <div className="page-stack">
      <PageHeader
        eyebrow="Accès rapides"
        title="Raccourcis"
        description="Rassemblez vos liens importants dans des panneaux accessibles en un clic."
        actions={<Button onClick={() => document.getElementById('shortcut-title')?.focus()}><Plus size={16} aria-hidden="true" /> Ajouter un raccourci</Button>}
      />

      {shortcutsQuery.isLoading ? <LoadingState label="Chargement des raccourcis…" /> : null}
      {shortcutsQuery.isError ? <ErrorState error={shortcutsQuery.error} onRetry={() => void shortcutsQuery.refetch()} title="Les raccourcis sont indisponibles" /> : null}
      {!shortcutsQuery.isLoading && !shortcutsQuery.isError ? (
        <ShortcutWorkspace shortcuts={shortcutsQuery.data ?? []} />
      ) : null}
    </div>
  )
}

function ShortcutWorkspace({ shortcuts }: { shortcuts: Shortcut[] }) {
  const createMutation = useCreateShortcutMutation()
  const updateMutation = useUpdateShortcutMutation()
  const deleteMutation = useDeleteShortcutMutation()
  const [form, setForm] = useState<ShortcutInput>(emptyShortcut)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [message, setMessage] = useState<string | null>(null)

  function editShortcut(shortcut: Shortcut) {
    setEditingId(shortcut.id)
    setForm({ title: shortcut.title, url: shortcut.url, description: shortcut.description })
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

  const formPending = createMutation.isPending || updateMutation.isPending
  const isValidationMessage = message?.includes('obligatoires') ?? false

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle>Vos panneaux</CardTitle>
          <CardDescription>Ouvrez un lien pour l’utiliser. Le serveur conserve le nombre d’ouvertures afin d’afficher les plus utilisés sur l’accueil.</CardDescription>
        </CardHeader>
        <CardContent>
          {shortcuts.length ? (
            <ShortcutGrid
              shortcuts={shortcuts}
              editable
              onEdit={editShortcut}
              onDelete={removeShortcut}
              className="shortcut-grid-manager"
            />
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
