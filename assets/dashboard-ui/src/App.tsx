import { lazy, Suspense } from 'react'
import { Link, Route, Routes } from 'react-router-dom'
import { AppShell } from './components/layout/AppShell'
import { LoadingState } from './components/feedback/LoadingState'

const DashboardPage = lazy(async () => {
  const module = await import('./features/dashboard/DashboardPage')
  return { default: module.DashboardPage }
})
const SearchPage = lazy(async () => {
  const module = await import('./features/search/SearchPage')
  return { default: module.SearchPage }
})
const SettingsPage = lazy(async () => {
  const module = await import('./features/settings/SettingsPage')
  return { default: module.SettingsPage }
})
const StatsPage = lazy(async () => {
  const module = await import('./features/stats/StatsPage')
  return { default: module.StatsPage }
})

function NotFoundPage() {
  return (
    <div className="not-found">
      <span className="eyebrow">404</span>
      <h1>Cette page n’existe pas.</h1>
      <p>Retournez à l’accueil du dashboard local.</p>
      <Link to="/" className="button button-default button-md">Retour à l’accueil</Link>
    </div>
  )
}

function AppRoutes() {
  return (
    <Suspense fallback={<LoadingState label="Chargement de l’interface…" />}>
      <Routes>
        <Route element={<AppShell />}>
          <Route index element={<DashboardPage />} />
          <Route path="search" element={<SearchPage />} />
          <Route path="stats" element={<StatsPage />} />
          <Route path="settings" element={<SettingsPage />} />
        </Route>
        <Route path="*" element={<NotFoundPage />} />
      </Routes>
    </Suspense>
  )
}

export default function App() {
  return <AppRoutes />
}
