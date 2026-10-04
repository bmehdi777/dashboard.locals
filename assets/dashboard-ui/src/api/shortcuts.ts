import { apiRequest, jsonBody } from './client'
import { normalizeShortcutFolders, normalizeShortcuts } from './normalize'
import type { Shortcut, ShortcutFolder, ShortcutFolderInput, ShortcutInput, ShortcutOrderItem, ShortcutSort } from './types'

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
  reorder: (items: ShortcutOrderItem[]) =>
    apiRequest<void>('/shortcuts/order', {
      method: 'PUT',
      body: jsonBody({
        items: items.map((item) => ({ id: item.id, folderId: item.folderId ?? null })),
      }),
    }),
}

export const shortcutFoldersApi = {
  list: async () =>
    normalizeShortcutFolders(await apiRequest<unknown>('/shortcut-folders')),
  create: (folder: ShortcutFolderInput) =>
    apiRequest<ShortcutFolder>('/shortcut-folders', {
      method: 'POST',
      body: jsonBody(folder),
    }),
  update: (id: string, folder: Partial<ShortcutFolderInput>) =>
    apiRequest<ShortcutFolder>(`/shortcut-folders/${encodeURIComponent(id)}`, {
      method: 'PATCH',
      body: jsonBody(folder),
    }),
  remove: (id: string) =>
    apiRequest<void>(`/shortcut-folders/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    }),
}
