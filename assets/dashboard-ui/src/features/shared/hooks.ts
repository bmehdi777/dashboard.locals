import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { healthApi, searchRootsApi, settingsApi, statsApi } from '../../api'
import type { SearchRoot, SearchRootInput, SettingsPatch, StatsFilters } from '../../api'

export const queryKeys = {
  health: ['health'] as const,
  settings: ['settings'] as const,
  roots: ['search-roots'] as const,
  stats: (filters: StatsFilters) => ['stats', filters] as const,
}

export function useHealthQuery() {
  return useQuery({
    queryKey: queryKeys.health,
    queryFn: healthApi.get,
    retry: false,
    refetchInterval: 30_000,
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
