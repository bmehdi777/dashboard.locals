export type HealthState = 'ok' | 'degraded' | 'error' | 'unknown'

export type OpenCodeState =
  | 'available'
  | 'missing'
  | 'absent'
  | 'stopped'
  | 'incompatible'
  | 'unauthenticated'
  | 'unknown'

export interface OpenCodeStatus {
  state: OpenCodeState
  version?: string
  message?: string
}

export interface HealthResponse {
  status: HealthState
  version?: string
  message?: string
  opencode?: OpenCodeStatus
  openCode?: OpenCodeStatus
}

export interface SearchRoot {
  id: string
  name: string
  path: string
  enabled: boolean
  createdAt?: string
  updatedAt?: string
}

export interface SearchPreferences {
  defaultRootId?: string
  respectGitignore: boolean
  ignoreBinary: boolean
  literal?: boolean
  maxResults?: number
  contextLines?: number
  timeout?: string
}

export interface EditorSettings {
  name: string
  command: string
  arguments: string[]
  placeholders?: string[]
}

export interface OpenCodeSettings {
  enabled: boolean
  serviceFile?: string
}

export type StatsTools = 'none' | 'summary' | 'detail'

export interface StatsSettings {
  timezone: string
  tools: StatsTools
  granularity: 'daily' | 'monthly'
}

export interface Settings {
  search: SearchPreferences
  editor: EditorSettings
  opencode: OpenCodeSettings
  stats: StatsSettings
}

export type SettingsPatch = Partial<Settings>

export interface SearchRootInput {
  name: string
  path: string
  enabled: boolean
}

export interface SearchRequest {
  rootId: string
  query: string
  respectGitignore: boolean
  ignoreBinary: boolean
  literal?: boolean
  maxResults?: number
  contextLines?: number
  timeoutMs?: number
}

export interface SearchResult {
  id?: string
  path: string
  line: number
  column?: number
  snippet: string
  context?: string[]
}

export interface SearchResponse {
  results: SearchResult[]
  total: number
  truncated?: boolean
  durationMs?: number
}

export interface OpenFileRequest {
  rootId: string
  path: string
  line?: number
  column?: number
}

export interface OpenFileResponse {
  opened: boolean
  message?: string
}

export interface StatsFilters {
  from?: string
  to?: string
  project?: string
  timezone?: string
  tools?: StatsTools
  granularity?: 'daily' | 'monthly'
}

export interface StatsSummary {
  sessions: number
  prompts: number
  steps: number
  inputTokens: number
  outputTokens: number
  reasoningTokens: number
  cacheReadTokens: number
  cacheWriteTokens: number
  cost: number
  activeDays: number
}

export interface DailyStats {
  date: string
  sessions: number
  prompts: number
  tokens: number
  cost: number
}

export interface ModelStats {
  model: string
  sessions: number
  tokens: number
  cost: number
}

export interface ToolStats {
  tool: string
  calls: number
  cost?: number
}

export interface StatsResponse {
  from?: string
  to?: string
  timezone?: string
  summary: StatsSummary
  daily: DailyStats[]
  models: ModelStats[]
  tools: ToolStats[]
  projects?: string[]
}

export interface StatsSyncResponse {
  synced: boolean
  records?: number
  message?: string
  rawCreated?: number
  rawExisting?: number
  aggregatesCreated?: number
  aggregatesUpdated?: number
}
