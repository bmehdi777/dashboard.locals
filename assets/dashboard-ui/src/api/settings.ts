import { apiRequest, jsonBody } from './client'
import { normalizeRoots, normalizeSettings } from './normalize'
import type {
  SearchRoot,
  SearchRootInput,
  SettingsPatch,
} from './types'

export const settingsApi = {
  get: async () => normalizeSettings(await apiRequest<unknown>('/settings')),
  update: async (patch: SettingsPatch) =>
    normalizeSettings(await apiRequest<unknown>('/settings', {
      method: 'PATCH',
      body: jsonBody(patch),
    })),
}

export const searchRootsApi = {
  list: async () => normalizeRoots(await apiRequest<unknown>('/search-roots')),
  create: (root: SearchRootInput) =>
    apiRequest<SearchRoot>('/search-roots', {
      method: 'POST',
      body: jsonBody(root),
    }),
  update: (id: string, root: Partial<SearchRootInput>) =>
    apiRequest<SearchRoot>(`/search-roots/${encodeURIComponent(id)}`, {
      method: 'PATCH',
      body: jsonBody(root),
    }),
  remove: (id: string) =>
    apiRequest<void>(`/search-roots/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    }),
}
