import { useEffect, useState } from 'react'

const userNameStorageKey = 'dashboard.locals.userName'
const userNameChangedEvent = 'dashboard.locals.userName.changed'
const appearanceStorageKey = 'dashboard.locals.appearance'
const appearanceChangedEvent = 'dashboard.locals.appearance.changed'
const sidebarCollapsedStorageKey = 'dashboard.locals.sidebarCollapsed'
const sidebarCollapsedChangedEvent = 'dashboard.locals.sidebarCollapsed.changed'
const maxUserNameLength = 80

export type AppearanceMode = 'dark' | 'light'
export type ColorTheme = 'violet' | 'blue' | 'green' | 'orange' | 'rose'

export interface AppearancePreferences {
  mode: AppearanceMode
  colorTheme: ColorTheme
}

export const defaultAppearance: AppearancePreferences = {
  mode: 'dark',
  colorTheme: 'violet',
}

function normalizeUserName(value: string | null | undefined): string {
  if (!value) return ''
  return Array.from(value.trim()).slice(0, maxUserNameLength).join('')
}

export function readUserName(): string {
  if (typeof window === 'undefined') return ''

  try {
    return normalizeUserName(window.localStorage.getItem(userNameStorageKey))
  } catch {
    return ''
  }
}

export function saveUserName(value: string): string {
  const normalized = normalizeUserName(value)

  if (typeof window === 'undefined') return normalized

  try {
    if (normalized) {
      window.localStorage.setItem(userNameStorageKey, normalized)
    } else {
      window.localStorage.removeItem(userNameStorageKey)
    }
    window.dispatchEvent(new Event(userNameChangedEvent))
  } catch {
    // Private browsing modes can make localStorage unavailable. The current
    // page can still use the normalized value even when persistence fails.
  }

  return normalized
}

export function useUserName(): string {
  const [userName, setUserName] = useState(readUserName)

  useEffect(() => {
    const refresh = () => setUserName(readUserName())
    window.addEventListener('storage', refresh)
    window.addEventListener(userNameChangedEvent, refresh)
    return () => {
      window.removeEventListener('storage', refresh)
      window.removeEventListener(userNameChangedEvent, refresh)
    }
  }, [])

  return userName
}

function normalizeAppearance(value: Partial<AppearancePreferences> | null | undefined): AppearancePreferences {
  return {
    mode: value?.mode === 'light' ? 'light' : 'dark',
    colorTheme:
      value?.colorTheme === 'blue' ||
      value?.colorTheme === 'green' ||
      value?.colorTheme === 'orange' ||
      value?.colorTheme === 'rose'
        ? value.colorTheme
        : 'violet',
  }
}

export function readAppearance(): AppearancePreferences {
  if (typeof window === 'undefined') return defaultAppearance

  try {
    const stored = window.localStorage.getItem(appearanceStorageKey)
    if (!stored) return defaultAppearance
    const parsed: unknown = JSON.parse(stored)
    if (!parsed || typeof parsed !== 'object') return defaultAppearance
    return normalizeAppearance(parsed as Partial<AppearancePreferences>)
  } catch {
    return defaultAppearance
  }
}

export function saveAppearance(value: Partial<AppearancePreferences>): AppearancePreferences {
  const appearance = normalizeAppearance(value)

  if (typeof window === 'undefined') return appearance

  try {
    window.localStorage.setItem(appearanceStorageKey, JSON.stringify(appearance))
    window.dispatchEvent(new Event(appearanceChangedEvent))
  } catch {
    // Keep the current appearance usable when localStorage is unavailable.
  }

  return appearance
}

export function useAppearance(): AppearancePreferences {
  const [appearance, setAppearance] = useState(readAppearance)

  useEffect(() => {
    const refresh = () => setAppearance(readAppearance())
    window.addEventListener('storage', refresh)
    window.addEventListener(appearanceChangedEvent, refresh)
    return () => {
      window.removeEventListener('storage', refresh)
      window.removeEventListener(appearanceChangedEvent, refresh)
    }
  }, [])

  return appearance
}

export function readSidebarCollapsed(): boolean {
  if (typeof window === 'undefined') return false

  try {
    return window.localStorage.getItem(sidebarCollapsedStorageKey) === 'true'
  } catch {
    return false
  }
}

export function saveSidebarCollapsed(collapsed: boolean): boolean {
  if (typeof window === 'undefined') return collapsed

  try {
    window.localStorage.setItem(sidebarCollapsedStorageKey, String(collapsed))
    window.dispatchEvent(new Event(sidebarCollapsedChangedEvent))
  } catch {
    // Keep the current layout usable when localStorage is unavailable.
  }

  return collapsed
}
