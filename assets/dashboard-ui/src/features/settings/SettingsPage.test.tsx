import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import userEvent from '@testing-library/user-event'
import { render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { SettingsPage } from './SettingsPage'

const settingsResponse = {
  search: {
    respectGitignore: true,
    includeBinary: false,
    literal: false,
    maxResults: 500,
    timeout: '10s',
  },
  editor: { name: '', command: '', arguments: [] },
  opencode: { enabled: true, serviceFile: '' },
  stats: { timezone: 'UTC', tools: 'summary', granularity: 'daily' },
}

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('SettingsPage', () => {
  it('affiche immédiatement une racine créée après la confirmation serveur', async () => {
    const roots: Array<Record<string, unknown>> = []
    const fetchMock = vi.fn<typeof fetch>().mockImplementation(async (input, init) => {
      const url = String(input)
      const method = init?.method ?? 'GET'
      if (url === '/api/v1/settings' && method === 'GET') {
        return new Response(JSON.stringify(settingsResponse), { status: 200 })
      }
      if (url.startsWith('/api/v1/search-roots') && method === 'GET') {
        return new Response(JSON.stringify({ data: roots }), { status: 200 })
      }
      if (url === '/api/v1/search-roots' && method === 'POST') {
        const body = JSON.parse(String(init?.body)) as Record<string, unknown>
        const root = { id: 'root-1', ...body }
        roots.push(root)
        return new Response(JSON.stringify(root), { status: 201 })
      }
      throw new Error(`Unexpected request: ${method} ${url}`)
    })
    vi.stubGlobal('fetch', fetchMock)

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const user = userEvent.setup()
    render(
      <QueryClientProvider client={queryClient}>
        <SettingsPage />
      </QueryClientProvider>,
    )

    await screen.findByText('Aucune racine configurée')
    await user.type(screen.getByPlaceholderText('Mon projet'), 'Projet')
    await user.type(screen.getByPlaceholderText('/home/moi/projets/mon-projet'), '/tmp/projet')
    await user.click(screen.getByRole('button', { name: /ajouter la racine/i }))

    await waitFor(() => expect(roots).toHaveLength(1))
    await waitFor(() => expect(queryClient.getQueryData(['search-roots', true])).toHaveLength(1))
    expect(await screen.findByText('Projet')).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/search-roots?includeDisabled=true', expect.anything())
  })
})
