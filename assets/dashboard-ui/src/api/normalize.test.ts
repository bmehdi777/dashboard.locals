import { describe, expect, it } from 'vitest'
import { normalizeHealth, normalizeRoots, normalizeStats } from './normalize'

describe('API normalizers', () => {
  it('accepte les enveloppes data et les champs snake_case', () => {
    expect(normalizeHealth({ data: { status: 'ok', openCode: { status: 'ready', version: '1.0' } } })).toMatchObject({
      status: 'ok',
      opencode: { state: 'available', version: '1.0' },
    })

    expect(normalizeRoots({ roots: [{ root_id: 'root-1', name: 'Projet', path: '/tmp/projet', enabled: false }] })).toEqual([
      expect.objectContaining({ id: 'root-1', name: 'Projet', path: '/tmp/projet', enabled: false }),
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
})
