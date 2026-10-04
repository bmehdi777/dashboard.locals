import { apiRequest } from './client'
import type { SearchHistoryEntry } from './types'

function normalizeHistory(value: unknown): SearchHistoryEntry[] {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return []
  const data = 'data' in value ? value.data : value
  if (!Array.isArray(data)) return []
  return data.filter((item): item is SearchHistoryEntry => {
    return Boolean(item && typeof item === 'object' && 'id' in item && 'query' in item)
  })
}

export const historyApi = {
  list: async (limit = 50) =>
    normalizeHistory(await apiRequest<unknown>(`/search-history?limit=${limit}`)),
  remove: (id: string) =>
    apiRequest<void>(`/search-history/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    }),
  clear: () =>
    apiRequest<void>('/search-history', {
      method: 'DELETE',
    }),
}
