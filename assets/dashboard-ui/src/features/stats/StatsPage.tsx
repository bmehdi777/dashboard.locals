import { useState, type FormEvent } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Activity,
  BrainCircuit,
  CalendarDays,
  Coins,
  DatabaseZap,
  RefreshCw,
  ServerCog,
  Sparkles,
  Workflow,
} from 'lucide-react'
import { getErrorMessage, statsApi } from '../../api'
import type { StatsFilters } from '../../api'
import { ActivityChart } from '../../components/stats/ActivityChart'
import { ModelBreakdown, ToolBreakdown } from '../../components/stats/BreakdownList'
import { MetricCard } from '../../components/stats/MetricCard'
import { EmptyState } from '../../components/feedback/EmptyState'
import { ErrorState } from '../../components/feedback/ErrorState'
import { LoadingState } from '../../components/feedback/LoadingState'
import { PageHeader } from '../../components/layout/PageHeader'
import { Alert } from '../../components/ui/Alert'
import { Button } from '../../components/ui/Button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../components/ui/Card'
import { Input } from '../../components/ui/Input'
import { Select } from '../../components/ui/Select'
import { useStatsQuery } from '../shared/hooks'
import { DEFAULT_STATS_FILTERS } from './defaults'

export function StatsPage() {
  const queryClient = useQueryClient()
  const [activeFilters, setActiveFilters] = useState<StatsFilters>(DEFAULT_STATS_FILTERS)
  const [draftFilters, setDraftFilters] = useState<StatsFilters>(DEFAULT_STATS_FILTERS)
  const [tools, setTools] = useState('')
  const [syncMessage, setSyncMessage] = useState<string | null>(null)
  const statsQuery = useStatsQuery(activeFilters)
  const syncMutation = useMutation({
    mutationFn: statsApi.sync,
    onSuccess: (response) => {
      setSyncMessage(response.message ?? `${response.records ?? 0} enregistrement(s) synchronisé(s).`)
      void queryClient.invalidateQueries({ queryKey: ['stats'] })
    },
  })

  function updateFilter<K extends keyof StatsFilters>(key: K, value: StatsFilters[K]) {
    setDraftFilters((current) => ({ ...current, [key]: value }))
  }

  function submitFilters(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setActiveFilters({
      ...draftFilters,
      tools: tools.split(',').map((tool) => tool.trim()).filter(Boolean),
    })
  }

  function resetFilters() {
    setDraftFilters(DEFAULT_STATS_FILTERS)
    setActiveFilters(DEFAULT_STATS_FILTERS)
    setTools('')
  }

  const stats = statsQuery.data
  const summary = stats?.summary

  return (
    <div className="page-stack">
      <PageHeader
        eyebrow="Usage IA"
        title="Statistiques"
        description="Analysez l’activité OpenCode conservée par le serveur local."
        actions={
          <Button onClick={() => { setSyncMessage(null); syncMutation.mutate() }} disabled={syncMutation.isPending}>
            <RefreshCw className={syncMutation.isPending ? 'spin' : undefined} size={16} aria-hidden="true" />
            {syncMutation.isPending ? 'Synchronisation…' : 'Synchroniser'}
          </Button>
        }
      />

      <Card className="filters-card">
        <CardHeader>
          <CardTitle>Filtres</CardTitle>
          <CardDescription>Les données sont demandées au serveur avec ces paramètres.</CardDescription>
        </CardHeader>
        <CardContent>
          <form className="filters-form" onSubmit={submitFilters}>
            <div className="form-field">
              <label htmlFor="stats-from">Du</label>
              <Input id="stats-from" type="date" value={draftFilters.from ?? ''} onChange={(event) => updateFilter('from', event.target.value || undefined)} />
            </div>
            <div className="form-field">
              <label htmlFor="stats-to">Au</label>
              <Input id="stats-to" type="date" value={draftFilters.to ?? ''} onChange={(event) => updateFilter('to', event.target.value || undefined)} />
            </div>
            <div className="form-field">
              <label htmlFor="stats-project">Projet</label>
              <Input id="stats-project" list="stats-project-list" value={draftFilters.project ?? ''} onChange={(event) => updateFilter('project', event.target.value || undefined)} placeholder="Tous les projets" />
              {stats?.projects?.length ? <datalist id="stats-project-list">{stats.projects.map((project) => <option value={project} key={project} />)}</datalist> : null}
            </div>
            <div className="form-field">
              <label htmlFor="stats-timezone">Fuseau horaire</label>
              <Input id="stats-timezone" value={draftFilters.timezone ?? ''} onChange={(event) => updateFilter('timezone', event.target.value || undefined)} placeholder="Europe/Paris" />
            </div>
            <div className="form-field">
              <label htmlFor="stats-granularity">Granularité</label>
              <Select id="stats-granularity" value={draftFilters.granularity ?? 'daily'} onChange={(event) => updateFilter('granularity', event.target.value as 'daily' | 'monthly')}>
                <option value="daily">Journalière</option>
                <option value="monthly">Mensuelle</option>
              </Select>
            </div>
            <div className="form-field">
              <label htmlFor="stats-tools">Outils</label>
              <Input id="stats-tools" value={tools} onChange={(event) => setTools(event.target.value)} placeholder="bash, edit, glob…" />
            </div>
            <div className="filters-actions">
              <Button type="button" variant="ghost" onClick={resetFilters}>Réinitialiser</Button>
              <Button type="submit"><Activity size={16} aria-hidden="true" /> Appliquer</Button>
            </div>
          </form>
        </CardContent>
      </Card>

      {syncMessage ? <Alert variant="success">{syncMessage}</Alert> : null}
      {syncMutation.isError ? <Alert variant="danger">{getErrorMessage(syncMutation.error)}</Alert> : null}

      {statsQuery.isLoading ? <LoadingState label="Chargement des statistiques…" /> : null}
      {statsQuery.isError ? <ErrorState error={statsQuery.error} onRetry={() => void statsQuery.refetch()} /> : null}

      {stats && summary ? (
        <>
          <section className="metric-grid" aria-label="Indicateurs principaux">
            <MetricCard label="Sessions" value={summary.sessions.toLocaleString('fr-FR')} detail={`${summary.activeDays} jour(s) actif(s)`} icon={<Workflow size={19} aria-hidden="true" />} tone="violet" />
            <MetricCard label="Tokens" value={(summary.inputTokens + summary.outputTokens + summary.reasoningTokens).toLocaleString('fr-FR')} detail={`${summary.inputTokens.toLocaleString('fr-FR')} entrants`} icon={<BrainCircuit size={19} aria-hidden="true" />} tone="blue" />
            <MetricCard label="Coût estimé" value={summary.cost.toLocaleString('fr-FR', { style: 'currency', currency: 'USD' })} detail={`${summary.cacheReadTokens.toLocaleString('fr-FR')} tokens en cache`} icon={<Coins size={19} aria-hidden="true" />} tone="orange" />
            <MetricCard label="Prompts" value={summary.prompts.toLocaleString('fr-FR')} detail={`${summary.steps.toLocaleString('fr-FR')} étapes`} icon={<Sparkles size={19} aria-hidden="true" />} tone="green" />
          </section>

          <section className="content-grid content-grid-wide">
            <Card className="chart-card">
              <CardHeader>
                <CardTitle>Activité quotidienne</CardTitle>
                <CardDescription>Volume de tokens sur la période sélectionnée.</CardDescription>
              </CardHeader>
              <CardContent>
                {stats.daily.length ? <ActivityChart data={stats.daily} /> : <EmptyState icon={<CalendarDays size={22} aria-hidden="true" />} title="Pas encore d’activité" description="Aucune donnée quotidienne ne correspond aux filtres choisis." />}
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle>Modèles utilisés</CardTitle>
                <CardDescription>Répartition par modèle.</CardDescription>
              </CardHeader>
              <CardContent>
                {stats.models.length ? <ModelBreakdown models={stats.models} /> : <EmptyState icon={<ServerCog size={22} aria-hidden="true" />} title="Aucun modèle" description="Les modèles apparaîtront après une synchronisation." />}
              </CardContent>
            </Card>
          </section>

          <Card>
            <CardHeader>
              <CardTitle>Outils</CardTitle>
              <CardDescription>Outils appelés pendant les sessions de la période.</CardDescription>
            </CardHeader>
            <CardContent>
              {stats.tools.length ? <ToolBreakdown tools={stats.tools} /> : <EmptyState icon={<DatabaseZap size={22} aria-hidden="true" />} title="Aucun outil" description="Aucune statistique d’outil n’est disponible." />}
            </CardContent>
          </Card>
        </>
      ) : null}
    </div>
  )
}
