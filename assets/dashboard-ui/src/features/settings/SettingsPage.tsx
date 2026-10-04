import { useState, type FormEvent } from 'react'
import { Check, Edit3, FolderPlus, Globe2, Save, Trash2, X } from 'lucide-react'
import { getErrorMessage } from '../../api'
import type { EditorSettings, SearchRoot, SearchRootInput, SearchPreferences, Settings, SyncSettings } from '../../api'
import { EmptyState } from '../../components/feedback/EmptyState'
import { ErrorState } from '../../components/feedback/ErrorState'
import { LoadingState } from '../../components/feedback/LoadingState'
import { PageHeader } from '../../components/layout/PageHeader'
import { Alert } from '../../components/ui/Alert'
import { Button } from '../../components/ui/Button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../components/ui/Card'
import { Checkbox } from '../../components/ui/Checkbox'
import { Input } from '../../components/ui/Input'
import { Select } from '../../components/ui/Select'
import { Textarea } from '../../components/ui/Textarea'
import {
  useCreateRootMutation,
  useDeleteRootMutation,
  useRootsQuery,
  useSettingsMutation,
  useSettingsQuery,
  useUpdateRootMutation,
} from '../shared/hooks'

const defaultSearch: SearchPreferences = {
  respectGitignore: true,
  ignoreBinary: true,
  maxResults: 500,
  contextLines: 0,
}

const defaultEditor: EditorSettings = {
  name: '',
  command: '',
  arguments: [],
  placeholders: [],
}

const defaultSync: SyncSettings = {
  enabled: true,
  intervalMinutes: 60,
  defaultGranularity: 'daily',
}

const defaultRoot: SearchRootInput = {
  name: '',
  path: '',
  enabled: true,
}

export function SettingsPage() {
  const settingsQuery = useSettingsQuery()
  const rootsQuery = useRootsQuery()

  return (
    <div className="page-stack">
      <PageHeader eyebrow="Configuration" title="Paramètres" description="Configurez vos racines de recherche et les préférences du dashboard." />
      {settingsQuery.isLoading || rootsQuery.isLoading ? <LoadingState label="Chargement des paramètres…" /> : null}
      {settingsQuery.isError ? <ErrorState error={settingsQuery.error} onRetry={() => void settingsQuery.refetch()} /> : null}
      {rootsQuery.isError ? <ErrorState error={rootsQuery.error} onRetry={() => void rootsQuery.refetch()} title="Les racines sont indisponibles" /> : null}
      {settingsQuery.data ? <SettingsWorkspace settings={settingsQuery.data} roots={rootsQuery.data ?? []} /> : null}
    </div>
  )
}

function SettingsWorkspace({ settings, roots }: { settings: Settings; roots: SearchRoot[] }) {
  const settingsMutation = useSettingsMutation()
  const createRootMutation = useCreateRootMutation()
  const updateRootMutation = useUpdateRootMutation()
  const deleteRootMutation = useDeleteRootMutation()
  const [search, setSearch] = useState<SearchPreferences>(settings.search ?? defaultSearch)
  const [editor, setEditor] = useState<EditorSettings>(settings.editor ?? defaultEditor)
  const [opencodeEnabled, setOpencodeEnabled] = useState(settings.opencode?.enabled ?? true)
  const [sync, setSync] = useState<SyncSettings>(settings.sync ?? defaultSync)
  const [rootForm, setRootForm] = useState<SearchRootInput>(defaultRoot)
  const [editingRootId, setEditingRootId] = useState<string | null>(null)
  const [settingsMessage, setSettingsMessage] = useState<string | null>(null)
  const [rootMessage, setRootMessage] = useState<string | null>(null)

  function submitSettings(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSettingsMessage(null)
    settingsMutation.mutate(
      {
        search,
        editor,
        opencode: { ...settings.opencode, enabled: opencodeEnabled },
        sync,
      },
      { onSuccess: () => setSettingsMessage('Les préférences ont été enregistrées.') },
    )
  }

  function editRoot(root: SearchRoot) {
    setEditingRootId(root.id)
    setRootForm({ name: root.name, path: root.path, enabled: root.enabled })
    setRootMessage(null)
  }

  function cancelRootEdit() {
    setEditingRootId(null)
    setRootForm(defaultRoot)
  }

  function submitRoot(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setRootMessage(null)
    const root = { name: rootForm.name.trim(), path: rootForm.path.trim(), enabled: rootForm.enabled }
    if (!root.name || !root.path) {
      setRootMessage('Le nom et le chemin sont obligatoires.')
      return
    }

    if (editingRootId) {
      updateRootMutation.mutate(
        { id: editingRootId, root },
        {
          onSuccess: () => { setRootMessage('La racine a été mise à jour.'); cancelRootEdit() },
          onError: (error) => setRootMessage(getErrorMessage(error)),
        },
      )
    } else {
      createRootMutation.mutate(root, {
        onSuccess: () => { setRootMessage('La racine a été ajoutée.'); setRootForm(defaultRoot) },
        onError: (error) => setRootMessage(getErrorMessage(error)),
      })
    }
  }

  function removeRoot(root: SearchRoot) {
    if (!window.confirm(`Supprimer la racine « ${root.name} » ?`)) return
    setRootMessage(null)
    deleteRootMutation.mutate(root.id, {
      onSuccess: () => setRootMessage('La racine a été supprimée.'),
      onError: (error) => setRootMessage(getErrorMessage(error)),
    })
  }

  const rootMutationPending = createRootMutation.isPending || updateRootMutation.isPending

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle>Racines de recherche</CardTitle>
          <CardDescription>Les chemins sont validés et restent sous le contrôle du serveur local.</CardDescription>
        </CardHeader>
        <CardContent>
          {rootMessage ? <Alert variant={rootMessage.includes('obligatoires') ? 'danger' : 'success'}>{rootMessage}</Alert> : null}
          <div className="root-list">
            {roots.map((root) => (
              <div className="root-row" key={root.id}>
                <div className="root-icon"><Globe2 size={18} aria-hidden="true" /></div>
                <div className="root-details"><strong>{root.name}</strong><code>{root.path}</code></div>
                <span className={`root-state ${root.enabled ? 'root-state-enabled' : ''}`}>{root.enabled ? 'Active' : 'Désactivée'}</span>
                <div className="root-actions">
                  <Button variant="ghost" size="icon" onClick={() => editRoot(root)} aria-label={`Modifier ${root.name}`}><Edit3 size={16} aria-hidden="true" /></Button>
                  <Button variant="ghost" size="icon" onClick={() => removeRoot(root)} disabled={deleteRootMutation.isPending} aria-label={`Supprimer ${root.name}`}><Trash2 size={16} aria-hidden="true" /></Button>
                </div>
              </div>
            ))}
          </div>
          {!roots.length ? <EmptyState icon={<FolderPlus size={22} aria-hidden="true" />} title="Aucune racine configurée" description="Ajoutez le premier dossier dans lequel vous souhaitez rechercher." /> : null}
          <form className="root-form" onSubmit={submitRoot}>
            <div className="root-form-title"><h3>{editingRootId ? 'Modifier une racine' : 'Ajouter une racine'}</h3>{editingRootId ? <Button type="button" variant="ghost" size="sm" onClick={cancelRootEdit}><X size={15} aria-hidden="true" /> Annuler</Button> : null}</div>
            <div className="form-grid form-grid-three">
              <div className="form-field"><label htmlFor="root-name">Nom</label><Input id="root-name" value={rootForm.name} onChange={(event) => setRootForm((current) => ({ ...current, name: event.target.value }))} placeholder="Mon projet" /></div>
              <div className="form-field form-field-wide"><label htmlFor="root-path">Chemin absolu</label><Input id="root-path" value={rootForm.path} onChange={(event) => setRootForm((current) => ({ ...current, path: event.target.value }))} placeholder="/home/moi/projets/mon-projet" /></div>
              <label className="checkbox-label root-enabled"><Checkbox checked={rootForm.enabled} onChange={(event) => setRootForm((current) => ({ ...current, enabled: event.target.checked }))} /><span>Racine active</span></label>
            </div>
            <Button type="submit" disabled={rootMutationPending}><FolderPlus size={16} aria-hidden="true" /> {editingRootId ? 'Enregistrer la racine' : 'Ajouter la racine'}</Button>
          </form>
        </CardContent>
      </Card>

      <form onSubmit={submitSettings} className="settings-form">
        <Card>
          <CardHeader><CardTitle>Recherche</CardTitle><CardDescription>Valeurs utilisées par défaut dans le formulaire de recherche.</CardDescription></CardHeader>
          <CardContent className="settings-section-content">
            <label className="checkbox-label"><Checkbox checked={search.respectGitignore} onChange={(event) => setSearch((current) => ({ ...current, respectGitignore: event.target.checked }))} /><span>Respecter les fichiers .gitignore par défaut</span></label>
            <label className="checkbox-label"><Checkbox checked={search.ignoreBinary} onChange={(event) => setSearch((current) => ({ ...current, ignoreBinary: event.target.checked }))} /><span>Ignorer les fichiers binaires par défaut</span></label>
            <div className="form-grid form-grid-two">
              <div className="form-field"><label htmlFor="max-results">Limite de résultats</label><Input id="max-results" type="number" min="1" max="10000" value={search.maxResults ?? ''} onChange={(event) => setSearch((current) => ({ ...current, maxResults: Number(event.target.value) || undefined }))} /></div>
              <div className="form-field"><label htmlFor="context-lines">Lignes de contexte</label><Input id="context-lines" type="number" min="0" max="20" value={search.contextLines ?? ''} onChange={(event) => setSearch((current) => ({ ...current, contextLines: Number(event.target.value) || 0 }))} /></div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader><CardTitle>Éditeur</CardTitle><CardDescription>Le serveur utilise cette configuration pour ouvrir les résultats avec des arguments séparés.</CardDescription></CardHeader>
          <CardContent className="settings-section-content">
            <div className="form-grid form-grid-two">
              <div className="form-field"><label htmlFor="editor-name">Nom</label><Input id="editor-name" value={editor.name} onChange={(event) => setEditor((current) => ({ ...current, name: event.target.value }))} placeholder="Visual Studio Code" /></div>
              <div className="form-field"><label htmlFor="editor-command">Exécutable</label><Input id="editor-command" value={editor.command} onChange={(event) => setEditor((current) => ({ ...current, command: event.target.value }))} placeholder="code" /></div>
            </div>
            <div className="form-field"><label htmlFor="editor-arguments">Arguments, un par ligne</label><Textarea id="editor-arguments" rows={4} value={editor.arguments.join('\n')} onChange={(event) => setEditor((current) => ({ ...current, arguments: event.target.value.split('\n').filter((argument) => argument.length > 0) }))} placeholder="--reuse-window\n{file}\n--goto\n{file}:{line}:{column}" /><span className="field-hint">Placeholders autorisés : {'{file}'}, {'{line}'}, {'{column}'}</span></div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader><CardTitle>OpenCode et synchronisation</CardTitle><CardDescription>L’absence d’OpenCode ne bloque pas le reste du dashboard. Aucun credential n’est affiché ici.</CardDescription></CardHeader>
          <CardContent className="settings-section-content">
            <label className="checkbox-label"><Checkbox checked={opencodeEnabled} onChange={(event) => setOpencodeEnabled(event.target.checked)} /><span>Activer la détection et la synchronisation OpenCode</span></label>
            <div className="form-grid form-grid-three">
              <div className="form-field"><label htmlFor="sync-interval">Intervalle (minutes)</label><Input id="sync-interval" type="number" min="1" value={sync.intervalMinutes ?? ''} onChange={(event) => setSync((current) => ({ ...current, intervalMinutes: Number(event.target.value) || undefined }))} /></div>
              <div className="form-field"><label htmlFor="sync-granularity">Granularité par défaut</label><Select id="sync-granularity" value={sync.defaultGranularity ?? 'daily'} onChange={(event) => setSync((current) => ({ ...current, defaultGranularity: event.target.value as 'daily' | 'monthly' }))}><option value="daily">Journalière</option><option value="monthly">Mensuelle</option></Select></div>
              <label className="checkbox-label root-enabled"><Checkbox checked={sync.enabled} onChange={(event) => setSync((current) => ({ ...current, enabled: event.target.checked }))} /><span>Synchronisation active</span></label>
            </div>
          </CardContent>
        </Card>

        <div className="settings-save-bar">
          {settingsMessage ? <Alert variant="success"><Check size={16} aria-hidden="true" /> {settingsMessage}</Alert> : null}
          {settingsMutation.isError ? <Alert variant="danger">{getErrorMessage(settingsMutation.error)}</Alert> : null}
          <Button type="submit" disabled={settingsMutation.isPending}><Save size={16} aria-hidden="true" /> {settingsMutation.isPending ? 'Enregistrement…' : 'Enregistrer les préférences'}</Button>
        </div>
      </form>
    </>
  )
}
