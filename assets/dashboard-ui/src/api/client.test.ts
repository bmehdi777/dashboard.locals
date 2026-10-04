import { afterEach, describe, expect, it, vi } from 'vitest'
import { historyApi } from './history'
import { searchApi } from './search'
import { settingsApi } from './settings'
import { statsApi } from './stats'

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('searchApi', () => {
  it('centralise les appels sur le préfixe API et normalise les résultats', async () => {
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(
      new Response(
        JSON.stringify({
          results: [{ path: 'src/App.tsx', line: 12, column: 4, match: 'useQuery' }],
          total: 1,
        }),
        { status: 200, headers: { 'Content-Type': 'application/json' } },
      ),
    )
    vi.stubGlobal('fetch', fetchMock)

    const response = await searchApi.search({
      rootId: 'root-1',
      query: 'useQuery',
      respectGitignore: true,
      ignoreBinary: true,
    })

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/search',
      expect.objectContaining({
        method: 'POST',
        headers: expect.objectContaining({ Accept: 'application/json' }),
      }),
    )
    const request = fetchMock.mock.calls[0]?.[1]
    expect(JSON.parse(String(request?.body))).toMatchObject({
      rootId: 'root-1',
      includeBinary: false,
    })
    expect(JSON.parse(String(request?.body))).not.toHaveProperty('ignoreBinary')
    expect(response.results[0]).toMatchObject({
      path: 'src/App.tsx',
      line: 12,
      column: 4,
      snippet: 'useQuery',
    })
  })

  it('retourne une erreur structurée lorsque le serveur échoue', async () => {
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(
      new Response(JSON.stringify({ error: { code: 'ROOT_NOT_FOUND', message: 'Racine inconnue.' } }), {
        status: 404,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    await expect(
      searchApi.search({
        rootId: 'missing',
        query: 'test',
        respectGitignore: true,
        ignoreBinary: true,
      }),
    ).rejects.toMatchObject({
      status: 404,
      code: 'ROOT_NOT_FOUND',
      message: 'Racine inconnue.',
    })
  })

  it('traduit les préférences UI vers le contrat strict du serveur', async () => {
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(
      new Response(JSON.stringify({
        search: { respectGitignore: true, includeBinary: false, literal: false, maxResults: 500, timeout: '10s' },
        editor: { name: '', command: '', arguments: [] },
        opencode: { enabled: true, serviceFile: '' },
        stats: { timezone: 'UTC', tools: 'summary', granularity: 'daily' },
      }),
        { status: 200, headers: { 'Content-Type': 'application/json' } }),
    )
    vi.stubGlobal('fetch', fetchMock)

    await settingsApi.update({
      search: { respectGitignore: true, ignoreBinary: false, maxResults: 500, contextLines: 4 },
      editor: { name: 'Code', command: 'code', arguments: ['{file}'], placeholders: ['{file}'] },
      opencode: { enabled: true, serviceFile: 'service.json' },
      stats: { timezone: 'Europe/Paris', tools: 'detail', granularity: 'monthly' },
    })

    const body = JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body)) as Record<string, unknown>
    expect(body.search).toEqual({ respectGitignore: true, includeBinary: true, maxResults: 500 })
    expect(body.editor).toEqual({ name: 'Code', command: 'code', arguments: ['{file}'] })
    expect(body.stats).toEqual({ timezone: 'Europe/Paris', tools: 'detail', granularity: 'monthly' })
    expect(body).not.toHaveProperty('sync')
  })

  it('formate les dates de formulaire et transmet les filtres de synchronisation', async () => {
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(
      new Response(JSON.stringify({ status: 'ok', rawCreated: 1, aggregatesCreated: 1 }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    await statsApi.sync({
      from: '2026-01-02',
      to: '2026-01-03',
      timezone: 'UTC',
      tools: 'summary',
      granularity: 'daily',
    })

    const body = JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body)) as Record<string, string>
    expect(body).toMatchObject({
      from: '2026-01-02T00:00:00Z',
      to: '2026-01-03T23:59:59Z',
      tools: 'summary',
    })
  })

  it('consomme les résultats de recherche au fil du flux NDJSON', async () => {
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(
      new Response(
        [
          JSON.stringify({ type: 'result', result: { path: 'first.go', line: 1, column: 1, excerpt: 'first' } }),
          JSON.stringify({ type: 'result', result: { path: 'second.go', line: 2, column: 3, excerpt: 'second' } }),
          JSON.stringify({ type: 'done', count: 2, truncated: false }),
        ].join('\n') + '\n',
        { status: 200, headers: { 'Content-Type': 'application/x-ndjson' } },
      ),
    )
    vi.stubGlobal('fetch', fetchMock)
    const events: Array<{ type: string }> = []

    await searchApi.stream(
      { rootId: 'root-1', query: 'needle', respectGitignore: true, ignoreBinary: true },
      undefined,
      (event) => events.push(event),
    )

    expect(events).toHaveLength(3)
    expect(events[0]).toMatchObject({ type: 'result', result: { path: 'first.go', snippet: 'first' } })
    expect(events[2]).toMatchObject({ type: 'done', count: 2 })
  })

  it('charge l’historique depuis l’enveloppe serveur', async () => {
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(
      new Response(JSON.stringify({ data: [{
        id: 'history-1', rootId: 'root-1', rootName: 'Projet', query: 'needle',
        literal: true, respectGitignore: true, includeBinary: false,
        resultCount: 3, truncated: false, status: 'completed', createdAt: '2026-01-01T12:00:00Z',
      }] }), { status: 200, headers: { 'Content-Type': 'application/json' } }),
    )
    vi.stubGlobal('fetch', fetchMock)

    const history = await historyApi.list()

    expect(history).toHaveLength(1)
    expect(history[0]).toMatchObject({ id: 'history-1', query: 'needle', resultCount: 3 })
  })
})
