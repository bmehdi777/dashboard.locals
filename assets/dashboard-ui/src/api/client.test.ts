import { afterEach, describe, expect, it, vi } from 'vitest'
import { searchApi } from './search'

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
})
