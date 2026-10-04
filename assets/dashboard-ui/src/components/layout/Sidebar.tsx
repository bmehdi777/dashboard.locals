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
import { cn } from '../../lib/utils'

const links = [
  { to: '/shortcuts', label: 'Raccourcis', icon: LayoutGrid },
  { to: '/', label: 'Vue d’ensemble', icon: LayoutDashboard, end: true },
  { to: '/search', label: 'Recherche', icon: Search },
  { to: '/stats', label: 'Statistiques', icon: BarChart3 },
  { to: '/settings', label: 'Paramètres', icon: Settings },
]

export function Sidebar({ open, onClose }: { open: boolean; onClose: () => void }) {
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
            <span>Recherche & usage IA</span>
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
        <div className="sidebar-footer">
          <span className="status-dot" aria-hidden="true" />
          <span>Serveur local</span>
        </div>
      </aside>
    </>
  )
}
