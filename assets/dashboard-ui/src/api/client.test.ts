import { afterEach, describe, expect, it, vi } from 'vitest'
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
})
