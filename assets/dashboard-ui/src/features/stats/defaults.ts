import type { StatsFilters } from '../../api'

export const DEFAULT_STATS_FILTERS: StatsFilters = {
  timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
  granularity: 'daily',
}
