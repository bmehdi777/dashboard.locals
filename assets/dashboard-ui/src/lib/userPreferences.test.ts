import { afterEach, describe, expect, it, vi } from 'vitest'
import { readAppearance, readSidebarCollapsed, readUserName, saveAppearance, saveSidebarCollapsed, saveUserName } from './userPreferences'

afterEach(() => {
  window.localStorage.clear()
  vi.restoreAllMocks()
})

describe('userPreferences', () => {
  it('enregistre le nom d’affichage uniquement dans le navigateur', () => {
    expect(readUserName()).toBe('')

    saveUserName('  Alice  ')

    expect(readUserName()).toBe('Alice')
    expect(window.localStorage.getItem('dashboard.locals.userName')).toBe('Alice')
  })

  it('réinitialise le nom lorsque la valeur est vide', () => {
    saveUserName('Alice')
    saveUserName('   ')

    expect(readUserName()).toBe('')
    expect(window.localStorage.getItem('dashboard.locals.userName')).toBeNull()
  })

  it('enregistre le mode et le thème de couleurs', () => {
    expect(readAppearance()).toEqual({ mode: 'dark', colorTheme: 'violet' })

    saveAppearance({ mode: 'light', colorTheme: 'blue' })

    expect(readAppearance()).toEqual({ mode: 'light', colorTheme: 'blue' })
    expect(window.localStorage.getItem('dashboard.locals.appearance')).toBe('{"mode":"light","colorTheme":"blue"}')
  })

  it('mémorise l’état replié du panneau latéral', () => {
    expect(readSidebarCollapsed()).toBe(false)

    saveSidebarCollapsed(true)

    expect(readSidebarCollapsed()).toBe(true)
    expect(window.localStorage.getItem('dashboard.locals.sidebarCollapsed')).toBe('true')
  })
})
