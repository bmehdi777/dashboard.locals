import type {
  DailyStats,
  HealthResponse,
  ModelStats,
  OpenFileResponse,
  OpenCodeState,
  OpenCodeStatus,
  SearchResponse,
  SearchResult,
  SearchRoot,
  Shortcut,
  SearchStreamEvent,
  Settings,
  StatsResponse,
  StatsSummary,
  StatsSyncResponse,
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
  if (state === 'missing' || state === 'absent' || state === 'not_found') return 'missing'
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

export function normalizeShortcuts(value: unknown): Shortcut[] {
  const source = unwrap(value)
  const shortcuts = isRecord(source) ? source.shortcuts : source

  return readArray(shortcuts).flatMap((item, index) => {
    if (!isRecord(item)) return []
    const id = readString(item.id) ?? `shortcut-${index}`
    const url = readString(item.url) ?? ''
    if (!url) return []
    return [
      {
        id,
        title: readString(item.title ?? item.name) ?? 'Raccourci sans nom',
        url,
        description: readString(item.description) ?? '',
        usageCount: asNumber(item.usageCount ?? item.usage_count),
        lastUsedAt: readString(item.lastUsedAt ?? item.last_used_at),
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
  const stats = isRecord(record.stats) ? record.stats : {}

  return {
    search: {
      defaultRootId: readString(search.defaultRootId ?? search.default_root_id),
      respectGitignore: readBoolean(
        search.respectGitignore ?? search.respect_gitignore,
        true,
      ),
      ignoreBinary:
        search.ignoreBinary !== undefined || search.ignore_binary !== undefined
          ? readBoolean(search.ignoreBinary ?? search.ignore_binary, true)
          : !readBoolean(search.includeBinary ?? search.include_binary, false),
      literal: readBoolean(search.literal, false),
      maxResults: asNumber(search.maxResults ?? search.max_results, 500),
      contextLines: asNumber(search.contextLines ?? search.context_lines, 0),
      timeout: readString(search.timeout),
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
      serviceFile: readString(opencode.serviceFile ?? opencode.service_file),
    },
    stats: {
      timezone: readString(stats.timezone) ?? 'UTC',
      tools:
        stats.tools === 'none' || stats.tools === 'detail' ? stats.tools : 'summary',
      granularity: stats.granularity === 'monthly' ? 'monthly' : 'daily',
    },
  }
}

function normalizeSearchResult(value: unknown, index: number): SearchResult | undefined {
  if (!isRecord(value)) return undefined

  const line = asNumber(value.line ?? value.lineNumber ?? value.line_number, 0)
  const snippet =
    readString(value.snippet ?? value.excerpt ?? value.match ?? value.text ?? value.lineText) ?? ''
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

export function normalizeSearchStreamEvent(
  value: unknown,
  index: number,
): SearchStreamEvent | undefined {
  if (!isRecord(value)) return undefined
  if (value.type === 'result') {
    const result = normalizeSearchResult(value.result, index)
    return result ? { type: 'result', result } : undefined
  }
  if (value.type === 'done') {
    return {
      type: 'done',
      count: asNumber(value.count),
      truncated: value.truncated === true,
    }
  }
  return undefined
}

export function normalizeOpenFile(value: unknown): OpenFileResponse {
  const source = unwrap(value)
  const record = isRecord(source) ? source : {}
  const status = readString(record.status)
  return {
    opened: record.opened === true || status === 'started' || status === 'opened',
    message: readString(record.message),
  }
}

export function normalizeStatsSync(value: unknown): StatsSyncResponse {
  const source = unwrap(value)
  const record = isRecord(source) ? source : {}
  const rawCreated = asNumber(record.rawCreated ?? record.raw_created)
  const rawExisting = asNumber(record.rawExisting ?? record.raw_existing)
  const aggregatesCreated = asNumber(
    record.aggregatesCreated ?? record.aggregates_created,
  )
  const aggregatesUpdated = asNumber(
    record.aggregatesUpdated ?? record.aggregates_updated,
  )
  const records = rawCreated + aggregatesCreated

  return {
    synced: readString(record.status) === 'ok' || record.synced === true,
    records,
    rawCreated,
    rawExisting,
    aggregatesCreated,
    aggregatesUpdated,
    message: readString(record.message),
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

function parseJSON(value: unknown): unknown {
  if (typeof value !== 'string') return value
  try {
    return JSON.parse(value) as unknown
  } catch {
    return undefined
  }
}

function dateKey(value: unknown): string {
  if (typeof value === 'number' && Number.isFinite(value)) {
    const milliseconds = value < 100_000_000_000 ? value * 1000 : value
    return new Date(milliseconds).toISOString().slice(0, 10)
  }
  const text = readString(value)
  if (!text) return ''
  const parsed = new Date(text)
  return Number.isNaN(parsed.getTime()) ? text.slice(0, 10) : parsed.toISOString().slice(0, 10)
}

function tokenTotal(value: unknown): number {
  if (!isRecord(value)) return asNumber(value)
  const nested = isRecord(value.tokens) ? value.tokens : value
  return (
    asNumber(nested.input) +
    asNumber(nested.output) +
    asNumber(nested.reasoning)
  )
}

function modelLabel(value: unknown): string {
  if (typeof value === 'string') return value
  if (!isRecord(value)) return 'Modèle inconnu'
  const model = value.model
  if (typeof model === 'string') return model
  if (isRecord(model)) {
    const id = readString(model.id)
    const provider = readString(model.providerID ?? model.providerId)
    if (provider && id) return `${provider}/${id}`
    if (id) return id
  }
  const directID = readString(value.id)
  const directProvider = readString(value.providerID ?? value.providerId)
  if (directProvider && directID) return `${directProvider}/${directID}`
  if (directID) return directID
  return readString(value.name) ?? 'Modèle inconnu'
}

function addModel(
  models: Map<string, ModelStats>,
  value: unknown,
  fallbackName?: string,
) {
  if (!isRecord(value)) return
  const model = modelLabel(value.model ?? value.name ?? fallbackName)
  const previous = models.get(model) ?? { model, sessions: 0, tokens: 0, cost: 0 }
  previous.sessions += asNumber(value.sessions)
  previous.tokens += tokenTotal(value.tokens ?? value.totalTokens ?? value)
  previous.cost += asNumber(value.cost)
  models.set(model, previous)
}

function collectModels(value: unknown, models: Map<string, ModelStats>) {
  const parsed = parseJSON(value)
  if (Array.isArray(parsed)) {
    parsed.forEach((item) => addModel(models, item))
    return
  }
  if (isRecord(parsed)) {
    if ('model' in parsed || 'name' in parsed || 'tokens' in parsed) {
      addModel(models, parsed)
      return
    }
    Object.entries(parsed).forEach(([name, item]) => addModel(models, item, name))
  }
}

function addTool(tools: Map<string, ToolStats>, name: string, calls: number, cost = 0) {
  const previous = tools.get(name) ?? { tool: name, calls: 0, cost: 0 }
  previous.calls += calls
  previous.cost = (previous.cost ?? 0) + cost
  tools.set(name, previous)
}

function collectTools(value: unknown, tools: Map<string, ToolStats>) {
  const parsed = parseJSON(value)
  if (Array.isArray(parsed)) {
    parsed.forEach((item) => {
      if (!isRecord(item)) return
      const name = readString(item.tool ?? item.name) ?? 'Outil inconnu'
      addTool(tools, name, asNumber(item.calls ?? item.count, 1), asNumber(item.cost))
    })
    return
  }
  if (!isRecord(parsed)) return
  if (typeof parsed.mode === 'string') addTool(tools, parsed.mode, 1)
  if ('tool' in parsed || 'name' in parsed) {
    const name = readString(parsed.tool ?? parsed.name) ?? 'Outil inconnu'
    addTool(tools, name, asNumber(parsed.calls ?? parsed.count, 1), asNumber(parsed.cost))
    return
  }
  Object.entries(parsed).forEach(([name, item]) => {
    if (typeof item === 'number') {
      addTool(tools, name, item)
    } else if (isRecord(item)) {
      addTool(tools, name, asNumber(item.calls ?? item.count, 1), asNumber(item.cost))
    }
  })
}

function normalizeAggregates(record: Record<string, unknown>): StatsResponse | undefined {
  const aggregates = readArray(record.aggregates)
  if (!aggregates.length) return undefined

  const summary: StatsSummary = {
    sessions: 0,
    prompts: 0,
    steps: 0,
    inputTokens: 0,
    outputTokens: 0,
    reasoningTokens: 0,
    cacheReadTokens: 0,
    cacheWriteTokens: 0,
    cost: 0,
    activeDays: 0,
  }
  const dailyMap = new Map<string, DailyStats>()
  const modelMap = new Map<string, ModelStats>()
  const toolMap = new Map<string, ToolStats>()

  aggregates.forEach((item) => {
    if (!isRecord(item)) return
    const inputTokens = asNumber(item.inputTokens ?? item.input_tokens)
    const outputTokens = asNumber(item.outputTokens ?? item.output_tokens)
    const reasoningTokens = asNumber(item.reasoningTokens ?? item.reasoning_tokens)
    const date = dateKey(item.periodStart ?? item.period_start)
    summary.sessions += asNumber(item.sessions)
    summary.prompts += asNumber(item.prompts)
    summary.steps += asNumber(item.steps)
    summary.inputTokens += inputTokens
    summary.outputTokens += outputTokens
    summary.reasoningTokens += reasoningTokens
    summary.cacheReadTokens += asNumber(item.cacheRead ?? item.cache_read)
    summary.cacheWriteTokens += asNumber(item.cacheWrite ?? item.cache_write)
    summary.cost += asNumber(item.cost)
    summary.activeDays += asNumber(item.activeDays ?? item.active_days)

    if (date) {
      const current = dailyMap.get(date) ?? { date, sessions: 0, prompts: 0, tokens: 0, cost: 0 }
      current.sessions += asNumber(item.sessions)
      current.prompts += asNumber(item.prompts)
      current.tokens += inputTokens + outputTokens + reasoningTokens
      current.cost += asNumber(item.cost)
      dailyMap.set(date, current)
    }
    collectModels(item.models, modelMap)
    collectTools(item.tools, toolMap)
  })

  if (summary.activeDays === 0) summary.activeDays = dailyMap.size
  const project = readString(record.project)
  return {
    from: readString(record.from),
    to: readString(record.to),
    timezone: undefined,
    summary,
    daily: [...dailyMap.values()].sort((left, right) => left.date.localeCompare(right.date)),
    models: [...modelMap.values()].sort((left, right) => right.tokens - left.tokens),
    tools: [...toolMap.values()].sort((left, right) => right.calls - left.calls),
    projects: project ? [project] : [],
  }
}

export function normalizeStats(value: unknown): StatsResponse {
  const source = unwrap(value)
  const record = isRecord(source) ? source : {}
  const aggregateStats = normalizeAggregates(record)
  if (aggregateStats) return aggregateStats
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
