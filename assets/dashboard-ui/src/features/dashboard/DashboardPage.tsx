import { ArrowRight, BarChart3, BrainCircuit, FolderSearch, Search, Settings2, Sparkles, Workflow } from 'lucide-react'
import { Link } from 'react-router-dom'
import { ActivityChart } from '../../components/stats/ActivityChart'
import { ModelBreakdown } from '../../components/stats/BreakdownList'
import { MetricCard } from '../../components/stats/MetricCard'
import { EmptyState } from '../../components/feedback/EmptyState'
import { ErrorState } from '../../components/feedback/ErrorState'
import { LoadingState } from '../../components/feedback/LoadingState'
import { PageHeader } from '../../components/layout/PageHeader'
import { OpenCodeBadge } from '../../components/feedback/StatusBadge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../components/ui/Card'
import { formatCost, formatNumber } from '../../lib/formatters'
import { useHealthQuery, useRootsQuery, useStatsQuery } from '../shared/hooks'
import { DEFAULT_STATS_FILTERS } from '../stats/defaults'

export function DashboardPage() {
  const healthQuery = useHealthQuery()
  const rootsQuery = useRootsQuery()
  const statsQuery = useStatsQuery(DEFAULT_STATS_FILTERS)
  const summary = statsQuery.data?.summary
  const opencode = healthQuery.data?.opencode ?? healthQuery.data?.openCode

  return (
    <div className="page-stack">
      <PageHeader
        eyebrow="Vue d’ensemble"
        title="Bonjour, développeur."
        description="Votre espace local pour explorer le code et comprendre votre usage IA."
        actions={<Link to="/search" className="button button-default button-md"><Search size={16} aria-hidden="true" /> Nouvelle recherche</Link>}
      />

      <section className="dashboard-hero">
        <div className="hero-copy">
          <span className="hero-kicker"><Sparkles size={15} aria-hidden="true" /> Espace local</span>
          <h2>Tout votre contexte, au même endroit.</h2>
          <p>Recherchez dans vos projets sans quitter votre environnement local et suivez l’activité de vos sessions OpenCode.</p>
          <div className="hero-actions">
            <Link to="/search" className="button button-secondary button-md"><FolderSearch size={16} aria-hidden="true" /> Explorer le code</Link>
            <Link to="/stats" className="button button-ghost button-md">Voir les statistiques <ArrowRight size={16} aria-hidden="true" /></Link>
          </div>
        </div>
        <div className="hero-orbit" aria-hidden="true">
          <div className="orbit orbit-one" />
          <div className="orbit orbit-two" />
          <div className="orbit-core"><BrainCircuit size={28} /></div>
        </div>
      </section>

      <section className="overview-status-grid" aria-label="État des services">
        <Card className="status-card">
          <CardContent>
            <div className="status-card-icon status-card-icon-green"><Workflow size={19} aria-hidden="true" /></div>
            <div>
              <p className="metric-label">Serveur local</p>
              <strong>{healthQuery.isLoading ? 'Vérification…' : healthQuery.isError ? 'Indisponible' : 'Opérationnel'}</strong>
              <span>{healthQuery.data?.version ? `Version ${healthQuery.data.version}` : 'API /api/v1'}</span>
            </div>
          </CardContent>
        </Card>
        <Card className="status-card">
          <CardContent>
            <div className="status-card-icon status-card-icon-violet"><BrainCircuit size={19} aria-hidden="true" /></div>
            <div>
              <p className="metric-label">OpenCode</p>
              {opencode ? <OpenCodeBadge state={opencode.state} /> : <strong>Non détecté</strong>}
              <span>{opencode?.message ?? 'Son absence ne bloque pas la recherche locale.'}</span>
            </div>
          </CardContent>
        </Card>
        <Card className="status-card">
          <CardContent>
            <div className="status-card-icon status-card-icon-blue"><FolderSearch size={19} aria-hidden="true" /></div>
            <div>
              <p className="metric-label">Racines actives</p>
              <strong>{rootsQuery.isLoading ? '…' : formatNumber(rootsQuery.data?.filter((root) => root.enabled).length ?? 0)}</strong>
              <span>{rootsQuery.data?.length ? 'Prêtes à être recherchées' : 'Configurez votre première racine'}</span>
            </div>
          </CardContent>
        </Card>
      </section>

      {statsQuery.isLoading ? <LoadingState label="Chargement du résumé statistique…" /> : null}
      {statsQuery.isError ? <ErrorState error={statsQuery.error} onRetry={() => void statsQuery.refetch()} title="Le résumé statistique est indisponible" /> : null}

      {summary ? (
        <section className="metric-grid" aria-label="Résumé des statistiques">
          <MetricCard label="Sessions" value={formatNumber(summary.sessions)} detail={`${formatNumber(summary.activeDays)} jours actifs`} icon={<Workflow size={19} aria-hidden="true" />} tone="violet" />
          <MetricCard label="Prompts" value={formatNumber(summary.prompts)} detail={`${formatNumber(summary.steps)} étapes`} icon={<Sparkles size={19} aria-hidden="true" />} tone="green" />
          <MetricCard label="Tokens" value={formatNumber(summary.inputTokens + summary.outputTokens + summary.reasoningTokens)} detail={`${formatNumber(summary.cacheReadTokens)} lus depuis le cache`} icon={<BrainCircuit size={19} aria-hidden="true" />} tone="blue" />
          <MetricCard label="Coût estimé" value={formatCost(summary.cost)} detail="Sur la période synchronisée" icon={<BarChart3 size={19} aria-hidden="true" />} tone="orange" />
        </section>
      ) : null}

      {statsQuery.data ? (
        <section className="content-grid content-grid-wide">
          <Card className="chart-card">
            <CardHeader>
              <CardTitle>Activité récente</CardTitle>
              <CardDescription>Les tokens traités jour après jour.</CardDescription>
            </CardHeader>
            <CardContent>
              {statsQuery.data.daily.length ? <ActivityChart data={statsQuery.data.daily} /> : <EmptyState icon={<BarChart3 size={22} aria-hidden="true" />} title="Pas encore de données" description="Synchronisez OpenCode pour alimenter ce graphique." />}
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>Modèles principaux</CardTitle>
              <CardDescription>Les modèles les plus utilisés.</CardDescription>
            </CardHeader>
            <CardContent>
              {statsQuery.data.models.length ? <ModelBreakdown models={statsQuery.data.models.slice(0, 4)} /> : <EmptyState icon={<BrainCircuit size={22} aria-hidden="true" />} title="Aucun modèle" description="Les données apparaîtront ici après synchronisation." />}
            </CardContent>
          </Card>
        </section>
      ) : null}

      {!rootsQuery.isLoading && !rootsQuery.isError && !rootsQuery.data?.length ? (
        <Card className="setup-card">
          <CardContent>
            <div className="setup-icon"><Settings2 size={22} aria-hidden="true" /></div>
            <div>
              <h2>Configurez votre espace</h2>
              <p>Ajoutez une racine de recherche pour commencer à explorer vos projets.</p>
            </div>
            <Link to="/settings" className="button button-outline button-sm">Ouvrir les paramètres <ArrowRight size={15} aria-hidden="true" /></Link>
          </CardContent>
        </Card>
      ) : null}
    </div>
  )
}
