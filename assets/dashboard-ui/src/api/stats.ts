import { apiRequest, jsonBody } from './client'
import { normalizeStats, normalizeStatsSync } from './normalize'
import type { StatsFilters } from './types'

function toRFC3339(value: string, endOfDay = false): string {
  if (/^\d{4}-\d{2}-\d{2}$/.test(value)) {
    return `${value}T${endOfDay ? '23:59:59' : '00:00:00'}Z`
  }
  return value
}

function toWireStatsRequest(filters: StatsFilters): Record<string, string> {
  const request: Record<string, string> = {}
  if (filters.from) request.from = toRFC3339(filters.from)
  if (filters.to) request.to = toRFC3339(filters.to, true)
  if (filters.project) request.project = filters.project
  if (filters.timezone) request.timezone = filters.timezone
  if (filters.granularity) request.granularity = filters.granularity
  if (filters.tools) request.tools = filters.tools
  return request
}

function toQuery(filters: StatsFilters): string {
  const params = new URLSearchParams()

  const request = toWireStatsRequest(filters)
  Object.entries(request).forEach(([key, value]) => params.set(key, value))

  const query = params.toString()
  return query ? `?${query}` : ''
}

export const statsApi = {
  get: async (filters: StatsFilters = {}) =>
    normalizeStats(await apiRequest<unknown>(`/stats${toQuery(filters)}`)),
  sync: async (filters: StatsFilters = {}) =>
    normalizeStatsSync(await apiRequest<unknown>('/stats/sync', {
      method: 'POST',
      body: jsonBody(toWireStatsRequest(filters)),
    })),
}
