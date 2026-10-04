import { apiRequest, jsonBody } from './client'
import { normalizeRoots, normalizeSettings } from './normalize'
import type {
  SearchRoot,
  SearchRootInput,
  SettingsPatch,
} from './types'

function toSettingsPatch(patch: SettingsPatch): Record<string, unknown> {
  const wire: Record<string, unknown> = {}

  if (patch.search) {
    wire.search = {
      ...(patch.search.respectGitignore === undefined
        ? {}
        : { respectGitignore: patch.search.respectGitignore }),
      ...(patch.search.ignoreBinary === undefined
        ? {}
        : { includeBinary: !patch.search.ignoreBinary }),
      ...(patch.search.literal === undefined ? {} : { literal: patch.search.literal }),
      ...(patch.search.maxResults === undefined
        ? {}
        : { maxResults: patch.search.maxResults }),
      ...(patch.search.timeout === undefined ? {} : { timeout: patch.search.timeout }),
    }
  }

  if (patch.editor) {
    wire.editor = {
      ...(patch.editor.name === undefined ? {} : { name: patch.editor.name }),
      ...(patch.editor.command === undefined ? {} : { command: patch.editor.command }),
      ...(patch.editor.arguments === undefined
        ? {}
        : { arguments: patch.editor.arguments }),
    }
  }

  if (patch.opencode) {
    wire.opencode = {
      ...(patch.opencode.enabled === undefined
        ? {}
        : { enabled: patch.opencode.enabled }),
      ...(patch.opencode.serviceFile === undefined
        ? {}
        : { serviceFile: patch.opencode.serviceFile }),
    }
  }

  if (patch.stats) {
    wire.stats = {
      ...(patch.stats.timezone === undefined
        ? {}
        : { timezone: patch.stats.timezone }),
      ...(patch.stats.tools === undefined ? {} : { tools: patch.stats.tools }),
      ...(patch.stats.granularity === undefined
        ? {}
        : { granularity: patch.stats.granularity }),
    }
  }

  return wire
}

export const settingsApi = {
  get: async () => normalizeSettings(await apiRequest<unknown>('/settings')),
  update: async (patch: SettingsPatch) =>
    normalizeSettings(await apiRequest<unknown>('/settings', {
      method: 'PATCH',
      body: jsonBody(toSettingsPatch(patch)),
    })),
}

export const searchRootsApi = {
  list: async (includeDisabled = true) =>
    normalizeRoots(
      await apiRequest<unknown>(
        `/search-roots${includeDisabled ? '?includeDisabled=true' : ''}`,
      ),
    ),
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
