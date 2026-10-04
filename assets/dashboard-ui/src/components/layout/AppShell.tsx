import { useEffect, useState } from 'react'
import { Outlet } from 'react-router-dom'
import { cn } from '../../lib/utils'
import { Header } from './Header'
import { Sidebar } from './Sidebar'

const mobileMediaQuery = '(max-width: 800px)'

function getIsMobileViewport() {
  return typeof window !== 'undefined' &&
    typeof window.matchMedia === 'function' &&
    window.matchMedia(mobileMediaQuery).matches
}

export function AppShell() {
  const [isMobileViewport, setIsMobileViewport] = useState(getIsMobileViewport)
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)

  useEffect(() => {
    if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return

    const mediaQuery = window.matchMedia(mobileMediaQuery)
    const updateViewport = () => setIsMobileViewport(mediaQuery.matches)
    updateViewport()
    if (typeof mediaQuery.addEventListener !== 'function') return

    mediaQuery.addEventListener('change', updateViewport)

    return () => mediaQuery.removeEventListener('change', updateViewport)
  }, [])

  function toggleSidebar() {
    if (isMobileViewport) {
      setSidebarOpen((open) => !open)
      return
    }

    setSidebarCollapsed((collapsed) => !collapsed)
  }

  const sidebarExpanded = isMobileViewport ? sidebarOpen : !sidebarCollapsed
  const menuLabel = isMobileViewport
    ? sidebarOpen ? 'Fermer le menu' : 'Ouvrir le menu'
    : sidebarCollapsed ? 'Afficher le panneau latéral' : 'Replier le panneau latéral'

  return (
    <div className={cn('app-shell', sidebarCollapsed && 'sidebar-collapsed')}>
      <Sidebar open={sidebarOpen} onClose={() => setSidebarOpen(false)} />
      <div className="app-main">
        <Header onMenu={toggleSidebar} sidebarExpanded={sidebarExpanded} menuLabel={menuLabel} />
        <main className="main-content">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
