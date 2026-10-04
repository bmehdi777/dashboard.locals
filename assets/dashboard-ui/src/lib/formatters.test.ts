import { describe, expect, it } from 'vitest'
import { formatCost, formatDuration, formatNumber } from './formatters'

describe('formatters', () => {
  it('formate les métriques avec les conventions françaises', () => {
    expect(formatNumber(123456)).toContain('123')
    expect(formatCost(12.5)).toContain('12')
    expect(formatDuration(850)).toBe('850 ms')
  })
})
