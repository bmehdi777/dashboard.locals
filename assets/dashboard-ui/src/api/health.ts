import { apiRequest } from './client'
import { normalizeHealth } from './normalize'

export const healthApi = {
  get: async () => normalizeHealth(await apiRequest<unknown>('/health')),
}
