import type {
  DailyStats,
  HealthResponse,
  ModelStats,
  OpenCodeState,
  OpenCodeStatus,
  SearchResponse,
  SearchResult,
  SearchRoot,
  Settings,
  StatsResponse,
  StatsSummary,
  ToolStats,
} from './types'
import { asNumber, isRecord } from '../lib/utils'

function unwrap(value: unknown): unknown {
  if (isRecord(value) && 'data' in value) return value.data
  return value
}

function readBoolean(value: unknown, fallback: boolean): boolean {
  return typeof value === 'boolean' ? value : fallback
}

function readString(value: unknown): string | undefined {
  return typeof value === 'string' && value.length > 0 ? value : undefined
}

function readArray(value: unknown): unknown[] {
  return Array.isArray(value) ? value : []
}

function normalizeOpenCodeState(value: unknown): OpenCodeState {
  if (typeof value !== 'string') return 'unknown'
  const state = value.toLowerCase()

  if (state === 'available' || state === 'ok' || state === 'ready' || state === 'healthy') return 'available'
  if (state === 'missing' || state === 'not_found') return 'missing'
  if (state === 'stopped' || state === 'offline') return 'stopped'
  if (state === 'incompatible') return 'incompatible'
  if (state === 'unauthenticated' || state === 'unauthorized') return 'unauthenticated'
  return 'unknown'
}

function normalizeOpenCode(value: unknown): OpenCodeStatus | undefined {
  if (!isRecord(value)) return undefined

  return {
    state: value.available === true ? 'available' : normalizeOpenCodeState(value.state ?? value.status),
    version: readString(value.version),
    message: readString(value.message),
  }
}

export function normalizeHealth(value: unknown): HealthResponse {
  const source = unwrap(value)
  if (!isRecord(source)) return { status: 'unknown' }

  const status = readString(source.status)?.toLowerCase()
  return {
    status:
      status === 'ok' || status === 'degraded' || status === 'error'
        ? status
        : 'unknown',
    version: readString(source.version),
    message: readString(source.message),
    opencode: normalizeOpenCode(source.opencode ?? source.openCode),
  }
}

export function normalizeRoots(value: unknown): SearchRoot[] {
  const source = unwrap(value)
  const roots = isRecord(source) ? source.roots : source

  return readArray(roots).flatMap((item, index) => {
    if (!isRecord(item)) return []
    const id = readString(item.id ?? item.rootId ?? item.root_id) ?? `root-${index}`
    const path = readString(item.path) ?? ''
    return [
      {
        id,
        name: readString(item.name) ?? (path || 'Racine sans nom'),
        path,
        enabled: readBoolean(item.enabled, true),
        createdAt: readString(item.createdAt ?? item.created_at),
        updatedAt: readString(item.updatedAt ?? item.updated_at),
      },
    ]
  })
}

export function normalizeSettings(value: unknown): Settings {
  const source = unwrap(value)
  const record = isRecord(source) ? source : {}
  const search = isRecord(record.search) ? record.search : {}
  const editor = isRecord(record.editor) ? record.editor : {}
  const opencode = isRecord(record.opencode ?? record.openCode)
    ? (record.opencode ?? record.openCode) as Record<string, unknown>
    : {}
  const sync = isRecord(record.sync) ? record.sync : {}

  return {
    search: {
      defaultRootId: readString(search.defaultRootId ?? search.default_root_id),
      respectGitignore: readBoolean(
        search.respectGitignore ?? search.respect_gitignore,
        true,
      ),
      ignoreBinary: readBoolean(
        search.ignoreBinary ?? search.ignore_binary,
        true,
      ),
      maxResults: asNumber(search.maxResults ?? search.max_results, 500),
      contextLines: asNumber(search.contextLines ?? search.context_lines, 0),
    },
    editor: {
      name: readString(editor.name) ?? 'Éditeur par défaut',
      command: readString(editor.command) ?? '',
      arguments: readArray(editor.arguments).filter(
        (argument): argument is string => typeof argument === 'string',
      ),
      placeholders: readArray(editor.placeholders).filter(
        (placeholder): placeholder is string => typeof placeholder === 'string',
      ),
    },
    opencode: {
      enabled: readBoolean(opencode.enabled, true),
      endpoint: readString(opencode.endpoint),
      project: readString(opencode.project),
      timezone: readString(opencode.timezone),
    },
    sync: {
      enabled: readBoolean(sync.enabled, true),
      intervalMinutes: asNumber(sync.intervalMinutes ?? sync.interval_minutes, 60),
      defaultGranularity:
        sync.defaultGranularity === 'monthly' ? 'monthly' : 'daily',
    },
  }
}

function normalizeSearchResult(value: unknown, index: number): SearchResult | undefined {
  if (!isRecord(value)) return undefined

  const line = asNumber(value.line ?? value.lineNumber ?? value.line_number, 0)
  const snippet =
    readString(value.snippet ?? value.match ?? value.text ?? value.lineText) ?? ''
  const path = readString(value.path ?? value.file ?? value.filePath) ?? ''
  if (!path && !snippet) return undefined

  return {
    id: readString(value.id) ?? `${path}:${line}:${index}`,
    path,
    line,
    column: asNumber(value.column ?? value.columnNumber ?? value.column_number, 0),
    snippet,
    context: readArray(value.context).filter(
      (lineValue): lineValue is string => typeof lineValue === 'string',
    ),
  }
}

export function normalizeSearch(value: unknown): SearchResponse {
  const source = unwrap(value)
  const record = isRecord(source) ? source : {}
  const rawResults = Array.isArray(source)
    ? source
    : record.results ?? record.matches ?? []
  const results = readArray(rawResults).flatMap((item, index) => {
    const result = normalizeSearchResult(item, index)
    return result ? [result] : []
  })

  return {
    results,
    total: asNumber(record.total ?? record.count, results.length),
    truncated: Boolean(record.truncated ?? record.limited),
    durationMs: asNumber(record.durationMs ?? record.duration_ms, 0),
  }
}

function normalizeSummary(value: unknown): StatsSummary {
  const source = isRecord(value) ? value : {}
  return {
    sessions: asNumber(source.sessions),
    prompts: asNumber(source.prompts),
    steps: asNumber(source.steps ?? source.stages),
    inputTokens: asNumber(source.inputTokens ?? source.input_tokens),
    outputTokens: asNumber(source.outputTokens ?? source.output_tokens),
    reasoningTokens: asNumber(source.reasoningTokens ?? source.reasoning_tokens),
    cacheReadTokens: asNumber(source.cacheReadTokens ?? source.cache_read_tokens),
    cacheWriteTokens: asNumber(source.cacheWriteTokens ?? source.cache_write_tokens),
    cost: asNumber(source.cost ?? source.totalCost ?? source.total_cost),
    activeDays: asNumber(source.activeDays ?? source.active_days),
  }
}

function normalizeDaily(value: unknown): DailyStats | undefined {
  if (!isRecord(value)) return undefined
  const date = readString(value.date ?? value.day) ?? ''
  if (!date) return undefined
  return {
    date,
    sessions: asNumber(value.sessions),
    prompts: asNumber(value.prompts),
    tokens: asNumber(value.tokens ?? value.totalTokens ?? value.total_tokens),
    cost: asNumber(value.cost),
  }
}

function normalizeModel(value: unknown): ModelStats | undefined {
  if (!isRecord(value)) return undefined
  const model = readString(value.model ?? value.name) ?? 'Modèle inconnu'
  return {
    model,
    sessions: asNumber(value.sessions),
    tokens: asNumber(value.tokens ?? value.totalTokens ?? value.total_tokens),
    cost: asNumber(value.cost),
  }
}

function normalizeTool(value: unknown): ToolStats | undefined {
  if (!isRecord(value)) return undefined
  const tool = readString(value.tool ?? value.name) ?? 'Outil inconnu'
  return {
    tool,
    calls: asNumber(value.calls ?? value.count),
    cost: asNumber(value.cost),
  }
}

export function normalizeStats(value: unknown): StatsResponse {
  const source = unwrap(value)
  const record = isRecord(source) ? source : {}
  const daily = readArray(record.daily ?? record.activity ?? record.byDay).flatMap(
    (item) => {
      const normalized = normalizeDaily(item)
      return normalized ? [normalized] : []
    },
  )
  const models = readArray(record.models ?? record.byModel).flatMap((item) => {
    const normalized = normalizeModel(item)
    return normalized ? [normalized] : []
  })
  const tools = readArray(record.tools ?? record.byTool).flatMap((item) => {
    const normalized = normalizeTool(item)
    return normalized ? [normalized] : []
  })

  return {
    from: readString(record.from),
    to: readString(record.to),
    timezone: readString(record.timezone),
    summary: normalizeSummary(record.summary ?? record.totals),
    daily,
    models,
    tools,
    projects: readArray(record.projects).filter(
      (project): project is string => typeof project === 'string',
    ),
  }
}
