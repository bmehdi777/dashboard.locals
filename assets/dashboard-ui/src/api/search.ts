import { apiRequest, apiStream, jsonBody } from './client'
import { ApiError } from './errors'
import { normalizeOpenFile, normalizeSearch, normalizeSearchStreamEvent } from './normalize'
import type {
  OpenFileRequest,
  SearchRequest,
  SearchStreamEvent,
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
  stream: async (
    request: SearchRequest,
    signal: AbortSignal | undefined,
    onEvent: (event: SearchStreamEvent) => void,
  ) => {
    let index = 0
    await apiStream<unknown>(
      '/search/stream',
      {
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
      },
      (value) => {
        if (isStreamError(value)) {
          throw new ApiError(
            value.message ?? 'La recherche a échoué.',
            200,
            { code: value.code },
          )
        }
        const event = normalizeSearchStreamEvent(value, index)
        if (!event) return
        if (event.type === 'result') index += 1
        onEvent(event)
      },
    )
  },
  openFile: async (request: OpenFileRequest) =>
    normalizeOpenFile(await apiRequest<unknown>('/files/open', {
      method: 'POST',
      body: jsonBody(request),
    })),
}

function isStreamError(value: unknown): value is { type: 'error'; code?: string; message?: string } {
  return typeof value === 'object' && value !== null && 'type' in value && value.type === 'error'
}
