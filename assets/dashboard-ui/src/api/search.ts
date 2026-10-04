import { apiRequest, jsonBody } from './client'
import { normalizeSearch } from './normalize'
import type {
  OpenFileRequest,
  OpenFileResponse,
  SearchRequest,
} from './types'

export const searchApi = {
  search: async (request: SearchRequest, signal?: AbortSignal) =>
    normalizeSearch(await apiRequest<unknown>('/search', {
      method: 'POST',
      body: jsonBody(request),
      signal,
    })),
  openFile: (request: OpenFileRequest) =>
    apiRequest<OpenFileResponse>('/files/open', {
      method: 'POST',
      body: jsonBody(request),
    }),
}
