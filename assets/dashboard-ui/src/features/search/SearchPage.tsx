import { useRef, useState, type FormEvent } from 'react'
import { useMutation } from '@tanstack/react-query'
import { Check, Clipboard, ExternalLink, FileSearch, Search as SearchIcon, X } from 'lucide-react'
import { getErrorMessage, searchApi } from '../../api'
import type { SearchRequest, SearchRoot, SearchPreferences } from '../../api'
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
import { useRootsQuery, useSettingsQuery } from '../shared/hooks'

export function SearchPage() {
  const rootsQuery = useRootsQuery()
  const settingsQuery = useSettingsQuery()
  const roots = rootsQuery.data?.filter((root) => root.enabled) ?? []

  return (
    <div className="page-stack">
      <PageHeader
        eyebrow="Code local"
        title="Recherche"
        description="Retrouvez rapidement une occurrence dans vos racines de code configurées."
      />

      {rootsQuery.isError ? <ErrorState error={rootsQuery.error} onRetry={() => void rootsQuery.refetch()} /> : null}
      {settingsQuery.isError ? <Alert variant="warning">Les préférences de recherche sont indisponibles. Les valeurs par défaut sont utilisées.</Alert> : null}
      {rootsQuery.isLoading || settingsQuery.isLoading ? <LoadingState label="Chargement des racines…" /> : null}

      {!rootsQuery.isLoading && !rootsQuery.isError && roots.length === 0 ? (
        <EmptyState
          icon={<FileSearch size={25} aria-hidden="true" />}
          title="Aucune racine disponible"
          description="Ajoutez ou activez une racine de recherche dans les paramètres pour commencer."
        />
      ) : null}

      {!rootsQuery.isLoading && roots.length > 0 ? (
        <SearchWorkspace
          roots={roots}
          preferences={settingsQuery.data?.search}
        />
      ) : null}
    </div>
  )
}

function SearchWorkspace({
  roots,
  preferences,
}: {
  roots: SearchRoot[]
  preferences?: SearchPreferences
}) {
  const [rootId, setRootId] = useState(preferences?.defaultRootId ?? roots[0]?.id ?? '')
  const [query, setQuery] = useState('')
  const [respectGitignore, setRespectGitignore] = useState(preferences?.respectGitignore ?? true)
  const [ignoreBinary, setIgnoreBinary] = useState(preferences?.ignoreBinary ?? true)
  const [literal, setLiteral] = useState(preferences?.literal ?? false)
  const [copiedPath, setCopiedPath] = useState<string | null>(null)
  const [actionMessage, setActionMessage] = useState<string | null>(null)
  const abortController = useRef<AbortController | null>(null)
  const lastRequest = useRef<SearchRequest | null>(null)

  const searchMutation = useMutation({
    mutationFn: ({ request, signal }: { request: SearchRequest; signal: AbortSignal }) =>
      searchApi.search(request, signal),
  })
  const openMutation = useMutation({
    mutationFn: searchApi.openFile,
    onSuccess: (response) => {
      setActionMessage(response.message ?? 'Le fichier a été envoyé à l’éditeur.')
    },
  })

  function runSearch(request: SearchRequest) {
    abortController.current?.abort()
    const controller = new AbortController()
    abortController.current = controller
    lastRequest.current = request
    searchMutation.mutate({ request, signal: controller.signal })
  }

  function submitSearch(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setActionMessage(null)
    const trimmedQuery = query.trim()
    if (!trimmedQuery || !rootId) return

    runSearch({
      rootId,
      query: trimmedQuery,
      respectGitignore,
      ignoreBinary,
      literal,
      maxResults: preferences?.maxResults ?? 500,
      contextLines: preferences?.contextLines ?? 0,
    })
  }

  function cancelSearch() {
    abortController.current?.abort()
    searchMutation.reset()
  }

  async function copyPath(path: string) {
    try {
      await navigator.clipboard.writeText(path)
      setCopiedPath(path)
      window.setTimeout(() => setCopiedPath(null), 1800)
    } catch {
      setActionMessage('La copie du chemin n’a pas pu être effectuée.')
    }
  }

  function openResult(path: string, line: number, column?: number) {
    setActionMessage(null)
    openMutation.mutate({ rootId, path, line, column })
  }

  const isAbortError = searchMutation.error instanceof DOMException && searchMutation.error.name === 'AbortError'
  const canSubmit = Boolean(query.trim() && rootId) && !searchMutation.isPending

  return (
    <>
      <Card className="search-card">
        <CardHeader>
          <CardTitle>Nouvelle recherche</CardTitle>
          <CardDescription>La recherche est exécutée par le serveur local et reste limitée à la racine sélectionnée.</CardDescription>
        </CardHeader>
        <CardContent>
          <form className="search-form" onSubmit={submitSearch}>
            <div className="form-field search-query-field">
              <label htmlFor="search-query">Motif ou texte à rechercher</label>
              <div className="input-with-icon">
                <SearchIcon size={18} aria-hidden="true" />
                <Input id="search-query" value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Ex. useQuery, TODO, nom de fonction…" autoComplete="off" />
              </div>
            </div>
            <div className="form-field">
              <label htmlFor="search-root">Racine</label>
              <Select id="search-root" value={rootId} onChange={(event) => setRootId(event.target.value)}>
                <option value="" disabled>Choisir une racine</option>
                {roots.map((root) => <option value={root.id} key={root.id}>{root.name}</option>)}
              </Select>
              {rootId ? <span className="field-hint">{roots.find((root) => root.id === rootId)?.path}</span> : null}
            </div>
            <div className="search-options" aria-label="Options de recherche">
              <label className="checkbox-label">
                <Checkbox checked={respectGitignore} onChange={(event) => setRespectGitignore(event.target.checked)} />
                <span>Respecter les fichiers .gitignore</span>
              </label>
              <label className="checkbox-label">
                <Checkbox checked={ignoreBinary} onChange={(event) => setIgnoreBinary(event.target.checked)} />
                <span>Ignorer les fichiers binaires</span>
              </label>
              <label className="checkbox-label">
                <Checkbox checked={literal} onChange={(event) => setLiteral(event.target.checked)} />
                <span>Recherche littérale</span>
              </label>
            </div>
            <div className="form-actions">
              {searchMutation.isPending ? <Button type="button" variant="outline" onClick={cancelSearch}><X size={16} aria-hidden="true" /> Annuler</Button> : null}
              <Button type="submit" disabled={!canSubmit}><SearchIcon size={16} aria-hidden="true" /> Rechercher</Button>
            </div>
          </form>
        </CardContent>
      </Card>

      {actionMessage ? <Alert variant="success">{actionMessage}</Alert> : null}
      {openMutation.isError ? <Alert variant="danger">{getErrorMessage(openMutation.error)}</Alert> : null}
      {searchMutation.isPending ? <LoadingState label="Recherche dans la racine sélectionnée…" /> : null}
      {searchMutation.isError && !isAbortError ? <ErrorState error={searchMutation.error} onRetry={() => { if (lastRequest.current) runSearch(lastRequest.current) }} title="La recherche a échoué" /> : null}

      {searchMutation.data && searchMutation.data.results.length === 0 ? (
        <EmptyState icon={<SearchIcon size={25} aria-hidden="true" />} title="Aucun résultat" description="Aucune occurrence ne correspond à ce motif dans la racine sélectionnée." />
      ) : null}

      {searchMutation.data && searchMutation.data.results.length > 0 ? (
        <Card>
          <CardHeader className="results-header">
            <div>
              <CardTitle>Résultats</CardTitle>
              <CardDescription>{searchMutation.data.total} occurrence(s) trouvée(s){searchMutation.data.truncated ? ' — affichage limité' : ''}</CardDescription>
            </div>
            {searchMutation.data.durationMs ? <span className="muted-text">{searchMutation.data.durationMs} ms</span> : null}
          </CardHeader>
          <CardContent className="results-content">
            <div className="result-list" role="list">
              {searchMutation.data.results.map((result) => {
                const resultId = result.id ?? `${result.path}:${result.line}`
                return (
                  <article className="result-item" key={resultId} role="listitem">
                    <div className="result-main">
                      <div className="result-path-row">
                        <code className="result-path">{result.path}</code>
                        <span className="result-position">L{result.line}{result.column ? `:C${result.column}` : ''}</span>
                      </div>
                      <pre className="result-snippet"><code>{result.snippet}</code></pre>
                    </div>
                    <div className="result-actions">
                      <Button variant="ghost" size="sm" onClick={() => void copyPath(result.path)} title="Copier le chemin">
                        {copiedPath === result.path ? <Check size={15} aria-hidden="true" /> : <Clipboard size={15} aria-hidden="true" />}
                        <span className="sr-only">Copier le chemin</span>
                      </Button>
                      <Button variant="outline" size="sm" onClick={() => openResult(result.path, result.line, result.column)} disabled={openMutation.isPending}>
                        <ExternalLink size={15} aria-hidden="true" /> Ouvrir
                      </Button>
                    </div>
                  </article>
                )
              })}
            </div>
          </CardContent>
        </Card>
      ) : null}
    </>
  )
}
