import { describe, expect, it } from 'vitest'
import { normalizeHealth, normalizeRoots, normalizeSettings, normalizeStats } from './normalize'

describe('API normalizers', () => {
  it('accepte les enveloppes data et les champs snake_case', () => {
    expect(normalizeHealth({ data: { status: 'ok', openCode: { status: 'ready', version: '1.0' } } })).toMatchObject({
      status: 'ok',
      opencode: { state: 'available', version: '1.0' },
    })

    expect(normalizeSettings({ search: { includeBinary: false } }).search.ignoreBinary).toBe(true)

    expect(normalizeRoots({ roots: [{ root_id: 'root-1', name: 'Projet', path: '/tmp/projet', enabled: false }] })).toEqual([
      expect.objectContaining({ id: 'root-1', name: 'Projet', path: '/tmp/projet', enabled: false }),
    ])
    expect(normalizeRoots({ data: [{ id: 'root-2', name: 'Second', path: '/tmp/second', enabled: true }] })).toEqual([
      expect.objectContaining({ id: 'root-2', name: 'Second', path: '/tmp/second', enabled: true }),
    ])
  })

  it('fournit des collections sûres pour une réponse statistique partielle', () => {
    const stats = normalizeStats({
      totals: { sessions: 2, input_tokens: 10, output_tokens: 20, total_cost: 0.42 },
      activity: [{ day: '2026-01-01', total_tokens: 30, cost: 0.42 }],
      byModel: [{ name: 'model-x', total_tokens: 30, sessions: 2 }],
    })

    expect(stats.summary).toMatchObject({ sessions: 2, inputTokens: 10, outputTokens: 20, cost: 0.42 })
    expect(stats.daily[0]).toMatchObject({ date: '2026-01-01', tokens: 30 })
    expect(stats.models[0]).toMatchObject({ model: 'model-x', tokens: 30 })
    expect(stats.tools).toEqual([])
  })

  it('convertit la vue d’agrégats du serveur en données utilisables par les graphiques', () => {
    const stats = normalizeStats({
      source: 'opencode',
      project: '/workspace/project',
      granularity: 'daily',
      from: '2026-01-01T00:00:00Z',
      to: '2026-01-02T00:00:00Z',
      aggregates: [{
        periodStart: '2026-01-01T00:00:00Z',
        periodEnd: '2026-01-02T00:00:00Z',
        sessions: 2,
        prompts: 3,
        steps: 4,
        inputTokens: 10,
        outputTokens: 20,
        reasoningTokens: 5,
        cacheRead: 6,
        cacheWrite: 7,
        cost: 0.5,
        activeDays: 1,
        models: [{ model: { id: 'model-x', providerID: 'provider' }, tokens: { input: 10, output: 20, reasoning: 5 }, cost: 0.5 }],
        tools: { bash: 4 },
      }],
    })

    expect(stats.summary).toMatchObject({ sessions: 2, prompts: 3, inputTokens: 10, outputTokens: 20, cost: 0.5 })
    expect(stats.daily[0]).toMatchObject({ date: '2026-01-01', tokens: 35 })
    expect(stats.models[0]).toMatchObject({ model: 'provider/model-x', tokens: 35 })
    expect(stats.tools[0]).toMatchObject({ tool: 'bash', calls: 4 })
    expect(stats.projects).toEqual(['/workspace/project'])
  })
})
