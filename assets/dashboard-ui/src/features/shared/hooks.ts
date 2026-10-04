import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { healthApi, historyApi, searchRootsApi, settingsApi, shortcutFoldersApi, shortcutsApi, statsApi } from '../../api'
import type { SearchHistoryEntry, SearchRoot, SearchRootInput, SettingsPatch, Shortcut, ShortcutFolderInput, ShortcutInput, ShortcutOrderItem, ShortcutSort, StatsFilters } from '../../api'

export const queryKeys = {
  health: ['health'] as const,
  settings: ['settings'] as const,
  roots: ['search-roots'] as const,
  history: ['search-history'] as const,
  shortcuts: ['shortcuts'] as const,
  shortcutFolders: ['shortcut-folders'] as const,
  stats: (filters: StatsFilters) => ['stats', filters] as const,
}

export function useHealthQuery() {
  return useQuery({
    queryKey: queryKeys.health,
    queryFn: healthApi.get,
    retry: false,
    refetchInterval: 10_000,
    staleTime: 15_000,
  })
}

export function useSettingsQuery() {
  return useQuery({
    queryKey: queryKeys.settings,
    queryFn: settingsApi.get,
    retry: false,
    staleTime: 30_000,
  })
}

export function useRootsQuery(includeDisabled = true) {
  return useQuery({
    queryKey: [...queryKeys.roots, includeDisabled] as const,
    queryFn: () => searchRootsApi.list(includeDisabled),
    retry: false,
    staleTime: 30_000,
  })
}

export function useStatsQuery(filters: StatsFilters, enabled = true) {
  return useQuery({
    queryKey: queryKeys.stats(filters),
    queryFn: () => statsApi.get(filters),
    enabled,
    retry: false,
    staleTime: 60_000,
  })
}

export function useShortcutsQuery(sort: ShortcutSort = 'recent', limit?: number) {
  return useQuery({
    queryKey: [...queryKeys.shortcuts, sort, limit] as const,
    queryFn: () => shortcutsApi.list(sort, limit),
    retry: false,
    staleTime: 30_000,
  })
}

export function useShortcutFoldersQuery() {
  return useQuery({
    queryKey: queryKeys.shortcutFolders,
    queryFn: shortcutFoldersApi.list,
    retry: false,
    staleTime: 30_000,
  })
}

export function useCreateShortcutMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (shortcut: ShortcutInput) => shortcutsApi.create(shortcut),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.shortcuts })
    },
  })
}

export function useUpdateShortcutMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, shortcut }: { id: string; shortcut: Partial<ShortcutInput> }) =>
      shortcutsApi.update(id, shortcut),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.shortcuts })
    },
  })
}

export function useCreateShortcutFolderMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (folder: ShortcutFolderInput) => shortcutFoldersApi.create(folder),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.shortcutFolders })
    },
  })
}

export function useUpdateShortcutFolderMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, folder }: { id: string; folder: Partial<ShortcutFolderInput> }) =>
      shortcutFoldersApi.update(id, folder),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.shortcutFolders })
    },
  })
}

export function useDeleteShortcutFolderMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => shortcutFoldersApi.remove(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.shortcutFolders })
      void queryClient.invalidateQueries({ queryKey: queryKeys.shortcuts })
    },
  })
}

export function useReorderShortcutsMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (items: ShortcutOrderItem[]) => shortcutsApi.reorder(items),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.shortcuts })
    },
  })
}

export function useDeleteShortcutMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => shortcutsApi.remove(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.shortcuts })
    },
  })
}

export function useRecordShortcutUseMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => shortcutsApi.use(id),
    onSuccess: (shortcut) => {
      queryClient.setQueriesData<Shortcut[]>({ queryKey: queryKeys.shortcuts }, (current) =>
        current?.map((item) => (item.id === shortcut.id ? shortcut : item)),
      )
      void queryClient.invalidateQueries({ queryKey: queryKeys.shortcuts })
    },
  })
}

export function useSearchHistoryQuery() {
  return useQuery({
    queryKey: queryKeys.history,
    queryFn: () => historyApi.list(50),
    retry: false,
    staleTime: 30_000,
  })
}

export function useDeleteSearchHistoryMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => historyApi.remove(id),
    onSuccess: (_result, id) => {
      queryClient.setQueryData<SearchHistoryEntry[]>(queryKeys.history, (current) =>
        current?.filter((entry) => entry.id !== id),
      )
      void queryClient.invalidateQueries({ queryKey: queryKeys.history })
    },
  })
}

export function useClearSearchHistoryMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: historyApi.clear,
    onSuccess: () => {
      queryClient.setQueryData<SearchHistoryEntry[]>(queryKeys.history, [])
      void queryClient.invalidateQueries({ queryKey: queryKeys.history })
    },
  })
}

export function useSettingsMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (patch: SettingsPatch) => settingsApi.update(patch),
    onSuccess: (settings) => {
      queryClient.setQueryData(queryKeys.settings, settings)
    },
  })
}

export function useCreateRootMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (root: SearchRootInput) => searchRootsApi.create(root),
    onSuccess: (root) => {
      queryClient.setQueriesData<SearchRoot[]>({ queryKey: queryKeys.roots }, (current) => {
        if (!current || current.some((item) => item.id === root.id)) return current
        return [...current, root].sort((left, right) => left.name.localeCompare(right.name))
      })
      void queryClient.invalidateQueries({ queryKey: queryKeys.roots })
    },
  })
}

export function useUpdateRootMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, root }: { id: string; root: Partial<SearchRootInput> }) =>
      searchRootsApi.update(id, root),
    onSuccess: (root) => {
      queryClient.setQueriesData<SearchRoot[]>({ queryKey: queryKeys.roots }, (current) =>
        current?.map((item) => (item.id === root.id ? root : item)),
      )
      void queryClient.invalidateQueries({ queryKey: queryKeys.roots })
    },
  })
}

export function useDeleteRootMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => searchRootsApi.remove(id),
    onSuccess: (_result, id) => {
      queryClient.setQueriesData<SearchRoot[]>({ queryKey: queryKeys.roots }, (current) =>
        current?.filter((item) => item.id !== id),
      )
      void queryClient.invalidateQueries({ queryKey: queryKeys.roots })
    },
  })
}
