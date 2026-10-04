import { apiRequest, jsonBody } from './client'
import { normalizeOpenFile, normalizeSearch } from './normalize'
import type {
  OpenFileRequest,
  SearchRequest,
} from './types'

export const searchApi = {
  search: async (request: SearchRequest, signal?: AbortSignal) =>
    normalizeSearch(await apiRequest<unknown>('/search', {
      method: 'POST',
      body: jsonBody({
        rootId: request.rootId,
        query: request.query,
        respectGitignore: request.respectGitignore,
        includeBinary: !request.ignoreBinary,
        ...(request.maxResults === undefined ? {} : { maxResults: request.maxResults }),
        ...(request.literal === undefined ? {} : { literal: request.literal }),
        ...(request.timeoutMs === undefined ? {} : { timeoutMs: request.timeoutMs }),
      }),
      signal,
    })),
  openFile: async (request: OpenFileRequest) =>
    normalizeOpenFile(await apiRequest<unknown>('/files/open', {
      method: 'POST',
      body: jsonBody(request),
    })),
}
