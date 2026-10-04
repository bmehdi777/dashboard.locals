import type { ColorTheme } from './userPreferences'

interface FaviconColors {
  primary: string
  secondary: string
  contrast: string
}

const faviconColors: Record<ColorTheme, FaviconColors> = {
  violet: { primary: '#b49cff', secondary: '#7663e8', contrast: '#15121f' },
  blue: { primary: '#8db7ff', secondary: '#3a68c5', contrast: '#0f1c36' },
  green: { primary: '#70ddb5', secondary: '#16815f', contrast: '#0d241d' },
  orange: { primary: '#f5bd78', secondary: '#b9651d', contrast: '#2b1b0c' },
  rose: { primary: '#f3a4c4', secondary: '#b94f7e', contrast: '#2b1220' },
}

function createFaviconSvg(colors: FaviconColors): string {
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 34 34"><defs><linearGradient id="brand-gradient" x1="0" y1="0" x2="1" y2="1"><stop stop-color="${colors.primary}"/><stop offset="1" stop-color="${colors.secondary}"/></linearGradient></defs><rect width="34" height="34" rx="10" fill="url(#brand-gradient)"/><g transform="translate(7.5 7.5) scale(0.7916667)" fill="none" stroke="${colors.contrast}" stroke-linecap="round" stroke-linejoin="round" stroke-width="2"><path d="m18 16 4-4-4-4"/><path d="m6 8-4 4 4 4"/><path d="m14.5 4-5 16"/></g></svg>`
}

export function updateFavicon(colorTheme: ColorTheme): void {
  if (typeof document === 'undefined') return

  const favicon = document.querySelector<HTMLLinkElement>('link[rel~="icon"]')
  if (!favicon) return

  const svg = createFaviconSvg(faviconColors[colorTheme])
  favicon.href = `data:image/svg+xml,${encodeURIComponent(svg)}`
}
