import { Menu, RefreshCw, Server } from 'lucide-react'
import { useQueryClient } from '@tanstack/react-query'
import { useHealthQuery, queryKeys } from '../../features/shared/hooks'
import { HealthBadge, OpenCodeBadge } from '../feedback/StatusBadge'
import { Button } from '../ui/Button'

export function Header({ onMenu }: { onMenu: () => void }) {
  const queryClient = useQueryClient()
  const health = useHealthQuery()
  const isRefreshing = health.isFetching
  const status = health.data?.status ?? 'unknown'
  const opencode = health.data?.opencode ?? health.data?.openCode

  function refresh() {
    void queryClient.invalidateQueries({ queryKey: queryKeys.health })
  }

  return (
    <header className="topbar">
      <Button className="mobile-menu" variant="ghost" size="icon" onClick={onMenu} aria-label="Ouvrir le menu">
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
