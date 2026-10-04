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

export function jsonBody(value: unknown): string {
  return JSON.stringify(value)
}
