import { useRef, useState, type FormEvent } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Check, ChevronLeft, ChevronRight, Clipboard, ExternalLink, FileSearch, History, LoaderCircle, Repeat2, Search as SearchIcon, Trash2, X } from 'lucide-react'
import { getErrorMessage, searchApi } from '../../api'
import type { SearchHistoryEntry, SearchRequest, SearchRoot, SearchPreferences, SearchScope, SearchStreamDoneEvent, SearchStreamEvent, SearchResult } from '../../api'
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
import { formatDate, formatNumber } from '../../lib/formatters'
import {
  queryKeys,
  useClearSearchHistoryMutation,
  useDeleteSearchHistoryMutation,
  useRootsQuery,
  useSearchHistoryQuery,
  useSettingsQuery,
} from '../shared/hooks'

const SEARCH_PAGE_SIZE = 25

export function SearchPage() {
  const rootsQuery = useRootsQuery()
  const settingsQuery = useSettingsQuery()
  const roots = rootsQuery.data?.filter((root) => root.enabled) ?? []

  return (
    <div className="page-stack">
      <PageHeader
        eyebrow="Code local"
        title="Recherche"
        description="Recherchez dans le contenu ou le nom des fichiers de vos racines de code configurées."
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
  const [searchIn, setSearchIn] = useState<SearchScope>('content')
  const [respectGitignore, setRespectGitignore] = useState(preferences?.respectGitignore ?? true)
  const [ignoreBinary, setIgnoreBinary] = useState(preferences?.ignoreBinary ?? true)
  const [literal, setLiteral] = useState(preferences?.literal ?? false)
  const [copiedPath, setCopiedPath] = useState<string | null>(null)
  const [actionMessage, setActionMessage] = useState<string | null>(null)
  const [streamResults, setStreamResults] = useState<SearchResult[]>([])
  const [streamSummary, setStreamSummary] = useState<SearchStreamDoneEvent | null>(null)
  const [resultPage, setResultPage] = useState(1)
  const queryClient = useQueryClient()
  const abortController = useRef<AbortController | null>(null)
  const lastRequest = useRef<SearchRequest | null>(null)
  const historyQuery = useSearchHistoryQuery()
  const deleteHistoryMutation = useDeleteSearchHistoryMutation()
  const clearHistoryMutation = useClearSearchHistoryMutation()

  const searchMutation = useMutation({
    mutationFn: ({ request, signal }: { request: SearchRequest; signal: AbortSignal }) =>
      searchApi.stream(request, signal, (event: SearchStreamEvent) => {
        if (event.type === 'result') {
          setStreamResults((current) => [...current, event.result])
        } else {
          setStreamSummary(event)
        }
      }),
    onMutate: () => {
      setStreamResults([])
      setStreamSummary(null)
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.history })
    },
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
    setResultPage(1)
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
      searchIn,
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

  function repeatHistory(entry: SearchHistoryEntry) {
    setRootId(entry.rootId)
    setQuery(entry.query)
    setSearchIn(entry.searchIn ?? 'content')
    setRespectGitignore(entry.respectGitignore)
    setIgnoreBinary(!entry.includeBinary)
    setLiteral(entry.literal)
    setActionMessage('Recherche restaurée dans le formulaire.')
  }

  function removeHistory(entry: SearchHistoryEntry) {
    deleteHistoryMutation.mutate(entry.id)
  }

  function clearHistory() {
    if (!window.confirm('Effacer tout l’historique des recherches ?')) return
    clearHistoryMutation.mutate()
  }

  const isAbortError = searchMutation.error instanceof DOMException && searchMutation.error.name === 'AbortError'
  const canSubmit = Boolean(query.trim() && rootId) && !searchMutation.isPending
  const hasCompletedEmptySearch = Boolean(
    streamSummary && streamResults.length === 0 && !searchMutation.isPending,
  )
  const resultPageCount = Math.max(1, Math.ceil(streamResults.length / SEARCH_PAGE_SIZE))
  const currentResultPage = Math.min(resultPage, resultPageCount)
  const firstVisibleResult = (currentResultPage - 1) * SEARCH_PAGE_SIZE
  const visibleResults = streamResults.slice(firstVisibleResult, firstVisibleResult + SEARCH_PAGE_SIZE)
  const firstResultNumber = firstVisibleResult + 1
  const lastResultNumber = Math.min(firstVisibleResult + SEARCH_PAGE_SIZE, streamResults.length)
  const resultNoun = searchIn === 'filename' ? 'fichier(s)' : 'occurrence(s)'

  return (
    <>
      <Card className="search-card">
        <CardHeader>
          <CardTitle>Nouvelle recherche</CardTitle>
           <CardDescription>Choisissez si le motif doit être recherché dans le contenu ou le nom des fichiers de la racine sélectionnée.</CardDescription>
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
            <div className="form-field">
              <label htmlFor="search-in">Rechercher dans</label>
              <Select id="search-in" value={searchIn} onChange={(event) => setSearchIn(event.target.value as SearchScope)}>
                <option value="content">Contenu des fichiers</option>
                <option value="filename">Nom du fichier (titre)</option>
              </Select>
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

      <Card className="history-card">
        <CardHeader className="results-header">
          <div>
            <CardTitle><span className="history-title"><History size={17} aria-hidden="true" /> Historique récent</span></CardTitle>
            <CardDescription>Les recherches terminées sont conservées par le serveur local.</CardDescription>
          </div>
          {historyQuery.data?.length ? (
            <Button variant="ghost" size="sm" onClick={clearHistory} disabled={clearHistoryMutation.isPending}>
              <Trash2 size={15} aria-hidden="true" /> Effacer tout
            </Button>
          ) : null}
        </CardHeader>
        <CardContent>
          {historyQuery.isLoading ? <LoadingState label="Chargement de l’historique…" /> : null}
          {historyQuery.isError ? <ErrorState error={historyQuery.error} onRetry={() => void historyQuery.refetch()} title="L’historique est indisponible" /> : null}
          {!historyQuery.isLoading && !historyQuery.isError && !historyQuery.data?.length ? (
            <EmptyState icon={<History size={22} aria-hidden="true" />} title="Aucune recherche enregistrée" description="Les recherches terminées apparaîtront ici." />
          ) : null}
          {historyQuery.data?.length ? (
            <div className="history-list" role="list">
              {historyQuery.data.map((entry) => {
                const canRepeat = roots.some((root) => root.id === entry.rootId)
                return (
                  <article className="history-row" key={entry.id} role="listitem">
                    <div className="history-main">
                      <code className="history-query">{entry.query}</code>
                      <span className="history-root">{entry.rootName}</span>
                    </div>
                    <div className="history-meta">
                      <span>{formatDate(entry.createdAt)}</span>
                      <span>{entry.searchIn === 'filename' ? 'Nom du fichier' : 'Contenu'}</span>
                      <span>{formatNumber(entry.resultCount)} résultat(s)</span>
                      {entry.truncated ? <span className="history-limited">Limité</span> : null}
                    </div>
                    <div className="history-actions">
                      <Button variant="outline" size="sm" onClick={() => repeatHistory(entry)} disabled={!canRepeat} title={canRepeat ? 'Restaurer cette recherche' : 'La racine n’est plus active'}>
                        <Repeat2 size={15} aria-hidden="true" /> Reprendre
                      </Button>
                      <Button variant="ghost" size="icon" onClick={() => removeHistory(entry)} disabled={deleteHistoryMutation.isPending} aria-label={`Supprimer la recherche ${entry.query}`}>
                        <Trash2 size={15} aria-hidden="true" />
                      </Button>
                    </div>
                  </article>
                )
              })}
            </div>
          ) : null}
        </CardContent>
      </Card>

      {actionMessage ? <Alert variant="success">{actionMessage}</Alert> : null}
      {openMutation.isError ? <Alert variant="danger">{getErrorMessage(openMutation.error)}</Alert> : null}
      {searchMutation.isPending ? (
        <div className="stream-progress" role="status" aria-live="polite">
          <LoaderCircle className="spin" size={18} aria-hidden="true" />
          <span>Recherche en cours…</span>
          <strong>{streamResults.length} {resultNoun} reçu(s)</strong>
        </div>
      ) : null}
      {searchMutation.isError && !isAbortError ? <ErrorState error={searchMutation.error} onRetry={() => { if (lastRequest.current) runSearch(lastRequest.current) }} title="La recherche a échoué" /> : null}

      {hasCompletedEmptySearch ? (
        <EmptyState
          icon={<SearchIcon size={25} aria-hidden="true" />}
          title="Aucun résultat"
          description={searchIn === 'filename'
            ? 'Aucun nom de fichier ne correspond à ce motif dans la racine sélectionnée.'
            : 'Aucune occurrence ne correspond à ce motif dans la racine sélectionnée.'}
        />
      ) : null}

      {streamResults.length > 0 ? (
        <Card>
          <CardHeader className="results-header">
            <div>
              <CardTitle>Résultats</CardTitle>
              <CardDescription>
                {streamSummary?.count ?? streamResults.length} {resultNoun} trouvé(s)
                {searchMutation.isPending ? ' — recherche en cours…' : ''}
                {streamSummary?.truncated ? ' — affichage limité' : ''}
                {!searchMutation.isPending && streamResults.length > 0 && resultPageCount > 1 ? ` — affichage ${firstResultNumber}–${lastResultNumber}` : ''}
              </CardDescription>
            </div>
            {searchMutation.isPending ? <span className="muted-text">Flux actif</span> : null}
          </CardHeader>
          <CardContent className="results-content">
            <div className="result-list" role="list">
              {visibleResults.map((result) => {
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
            {!searchMutation.isPending && resultPageCount > 1 ? (
              <nav className="results-pagination" aria-label="Pagination des résultats">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setResultPage((page) => Math.max(1, page - 1))}
                  disabled={currentResultPage === 1}
                  aria-label="Page précédente"
                >
                  <ChevronLeft size={15} aria-hidden="true" />
                  <span>Précédent</span>
                </Button>
                <span className="results-page-status" aria-live="polite">
                  Page {currentResultPage} sur {resultPageCount}
                </span>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setResultPage((page) => Math.min(resultPageCount, page + 1))}
                  disabled={currentResultPage === resultPageCount}
                  aria-label="Page suivante"
                >
                  <span>Suivant</span>
                  <ChevronRight size={15} aria-hidden="true" />
                </Button>
              </nav>
            ) : null}
          </CardContent>
        </Card>
      ) : null}
    </>
  )
}
