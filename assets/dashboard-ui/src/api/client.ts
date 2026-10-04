import { ApiError, type ApiErrorPayload } from './errors'

export const API_PREFIX = '/api/v1'

interface RequestOptions extends RequestInit {
  signal?: AbortSignal
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

async function readPayload(response: Response): Promise<unknown> {
  const text = await response.text()

  if (!text) {
    return undefined
  }

  try {
    return JSON.parse(text) as unknown
  } catch {
    return text
  }
}

function getErrorPayload(payload: unknown): ApiErrorPayload {
  if (!isRecord(payload)) {
    return {}
  }

  const nested = isRecord(payload.error) ? payload.error : payload
  return {
    code: typeof nested.code === 'string' ? nested.code : undefined,
    message: typeof nested.message === 'string' ? nested.message : undefined,
    details: nested.details,
  }
}

function getErrorMessage(payload: unknown, status: number): string {
  const error = getErrorPayload(payload)
  return error.message ?? `La requête a échoué (${status}).`
}

export async function apiRequest<T>(
  path: string,
  options: RequestOptions = {},
): Promise<T> {
  const response = await fetch(`${API_PREFIX}${path}`, {
    ...options,
    headers: {
      Accept: 'application/json',
      ...(options.body ? { 'Content-Type': 'application/json' } : {}),
      ...options.headers,
    },
  })
  const payload = await readPayload(response)

  if (!response.ok) {
    const errorPayload = getErrorPayload(payload)
    throw new ApiError(
      getErrorMessage(payload, response.status),
      response.status,
      errorPayload,
    )
  }

  return payload as T
}

export async function apiStream<T>(
  path: string,
  options: RequestOptions,
  onEvent: (event: T) => void | Promise<void>,
): Promise<void> {
  const response = await fetch(`${API_PREFIX}${path}`, {
    ...options,
    headers: {
      Accept: 'application/x-ndjson, application/json',
      ...(options.body ? { 'Content-Type': 'application/json' } : {}),
      ...options.headers,
    },
  })

  if (!response.ok) {
    const payload = await readPayload(response)
    const errorPayload = getErrorPayload(payload)
    throw new ApiError(
      getErrorMessage(payload, response.status),
      response.status,
      errorPayload,
    )
  }

  if (!response.body) {
    throw new ApiError('Le serveur n’a pas fourni de flux de recherche.', response.status)
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''

  async function consumeLine(line: string) {
    const trimmed = line.trim()
    if (!trimmed) return
    let event: T
    try {
      event = JSON.parse(trimmed) as T
    } catch {
      throw new ApiError('Le flux de recherche contient une réponse invalide.', response.status)
    }
    await onEvent(event)
  }

  while (true) {
    const { done, value } = await reader.read()
    buffer += decoder.decode(value, { stream: !done })
    const lines = buffer.split('\n')
    buffer = lines.pop() ?? ''
    for (const line of lines) {
      await consumeLine(line)
    }
    if (done) break
  }
  if (buffer) await consumeLine(buffer)
}

export function jsonBody(value: unknown): string {
  return JSON.stringify(value)
}
