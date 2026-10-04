import {
  BarChart3,
  Code2,
  LayoutDashboard,
  LayoutGrid,
  Search,
  Settings,
  X,
} from 'lucide-react'
import { NavLink } from 'react-router-dom'
import { useHealthQuery } from '../../features/shared/hooks'
import type { HealthState } from '../../api'
import { cn } from '../../lib/utils'

const links = [
  { to: '/', label: 'Vue d’ensemble', icon: LayoutDashboard, end: true },
  { to: '/shortcuts', label: 'Raccourcis', icon: LayoutGrid },
  { to: '/search', label: 'Recherche', icon: Search },
  { to: '/stats', label: 'Statistiques', icon: BarChart3 },
  { to: '/settings', label: 'Paramètres', icon: Settings },
]

export function Sidebar({ open, onClose }: { open: boolean; onClose: () => void }) {
  const health = useHealthQuery()
  const status: HealthState = health.isError ? 'error' : health.data?.status ?? 'unknown'
  const statusLabels: Record<HealthState, string> = {
    ok: 'Serveur local',
    degraded: 'Serveur local dégradé',
    error: 'Serveur local indisponible',
    unknown: 'Serveur local — état inconnu',
  }

  return (
    <>
      {open ? <button className="sidebar-overlay" aria-label="Fermer le menu" onClick={onClose} /> : null}
      <aside id="primary-sidebar" className={cn('sidebar', open && 'sidebar-open')}>
        <div className="sidebar-brand">
          <div className="brand-mark" aria-hidden="true">
            <Code2 size={19} />
          </div>
          <div>
            <strong>dashboard.locals</strong>
            <span>Recherche &amp; utilitaire</span>
          </div>
          <button className="mobile-close" onClick={onClose} aria-label="Fermer le menu">
            <X size={18} aria-hidden="true" />
          </button>
        </div>
        <nav className="sidebar-nav" aria-label="Navigation principale">
          <p className="nav-label">Espace de travail</p>
          {links.map(({ to, label, icon: Icon, end }) => (
            <NavLink
              key={to}
              to={to}
              end={end}
              onClick={onClose}
              className={({ isActive }) => cn('nav-link', isActive && 'nav-link-active')}
            >
              <Icon size={18} aria-hidden="true" />
              <span>{label}</span>
            </NavLink>
          ))}
        </nav>
        <div className={cn('sidebar-footer', `sidebar-footer-${status}`)} aria-live="polite">
          <span className={cn('status-dot', `status-dot-${status}`)} aria-hidden="true" />
          <span>{statusLabels[status]}</span>
        </div>
      </aside>
    </>
  )
}
