import { Menu, RefreshCw, Server } from 'lucide-react'
import { useQueryClient } from '@tanstack/react-query'
import { useHealthQuery, queryKeys } from '../../features/shared/hooks'
import { HealthBadge, OpenCodeBadge } from '../feedback/StatusBadge'
import { Button } from '../ui/Button'

interface HeaderProps {
  onMenu: () => void
  sidebarExpanded: boolean
  menuLabel: string
}

export function Header({ onMenu, sidebarExpanded, menuLabel }: HeaderProps) {
  const queryClient = useQueryClient()
  const health = useHealthQuery()
  const isRefreshing = health.isFetching
  const status = health.isError ? 'error' : health.data?.status ?? 'unknown'
  const opencode = health.data?.opencode ?? health.data?.openCode

  function refresh() {
    void queryClient.invalidateQueries({ queryKey: queryKeys.health })
  }

  return (
    <header className="topbar">
      <Button
        className="sidebar-toggle"
        variant="ghost"
        size="icon"
        onClick={onMenu}
        aria-label={menuLabel}
        aria-expanded={sidebarExpanded}
        aria-controls="primary-sidebar"
        title={menuLabel}
      >
        <Menu size={20} aria-hidden="true" />
      </Button>
      <div className="topbar-context">
        <Server size={16} aria-hidden="true" />
        <span>Instance locale</span>
      </div>
      <div className="topbar-status">
        <HealthBadge status={status} />
        {opencode ? <OpenCodeBadge state={opencode.state} /> : null}
        <Button variant="ghost" size="icon" onClick={refresh} disabled={isRefreshing} aria-label="Actualiser le statut" title="Actualiser le statut">
          <RefreshCw className={isRefreshing ? 'spin' : undefined} size={17} aria-hidden="true" />
        </Button>
      </div>
    </header>
  )
}
