import { apiRequest, jsonBody } from './client'
import { normalizeShortcuts } from './normalize'
import type { Shortcut, ShortcutInput, ShortcutSort } from './types'

export const shortcutsApi = {
  list: async (sort: ShortcutSort = 'recent', limit?: number) => {
    const query = new URLSearchParams({ sort })
    if (limit !== undefined) query.set('limit', String(limit))
    return normalizeShortcuts(
      await apiRequest<unknown>(`/shortcuts?${query.toString()}`),
    )
  },
  create: (shortcut: ShortcutInput) =>
    apiRequest<Shortcut>('/shortcuts', {
      method: 'POST',
      body: jsonBody(shortcut),
    }),
  update: (id: string, shortcut: Partial<ShortcutInput>) =>
    apiRequest<Shortcut>(`/shortcuts/${encodeURIComponent(id)}`, {
      method: 'PATCH',
      body: jsonBody(shortcut),
    }),
  remove: (id: string) =>
    apiRequest<void>(`/shortcuts/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    }),
  use: (id: string) =>
    apiRequest<Shortcut>(`/shortcuts/${encodeURIComponent(id)}/use`, {
      method: 'POST',
    }),
}
