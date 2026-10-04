import { apiRequest } from './client'
import { normalizeStats } from './normalize'
import type {
  StatsFilters,
  StatsSyncResponse,
} from './types'

function toQuery(filters: StatsFilters): string {
  const params = new URLSearchParams()

  if (filters.from) params.set('from', filters.from)
  if (filters.to) params.set('to', filters.to)
  if (filters.project) params.set('project', filters.project)
  if (filters.timezone) params.set('timezone', filters.timezone)
  if (filters.granularity) params.set('granularity', filters.granularity)
  if (filters.tools?.length) params.set('tools', filters.tools.join(','))

  const query = params.toString()
  return query ? `?${query}` : ''
}

export const statsApi = {
  get: async (filters: StatsFilters = {}) =>
    normalizeStats(await apiRequest<unknown>(`/stats${toQuery(filters)}`)),
  sync: () =>
    apiRequest<StatsSyncResponse>('/stats/sync', {
      method: 'POST',
    }),
}
