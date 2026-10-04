import { afterEach, describe, expect, it } from 'vitest'
import { updateFavicon } from './favicon'

afterEach(() => {
  document.head.innerHTML = ''
})

describe('favicon', () => {
  it('utilise les couleurs du thème sélectionné', () => {
    const favicon = document.createElement('link')
    favicon.rel = 'icon'
    document.head.appendChild(favicon)

    updateFavicon('blue')

    expect(decodeURIComponent(favicon.href)).toContain('stop-color="#8db7ff"')
    expect(decodeURIComponent(favicon.href)).toContain('stop-color="#3a68c5"')
    expect(decodeURIComponent(favicon.href)).toContain('stroke="#0f1c36"')
  })

  it('ne fait rien si aucun favicon n’est déclaré', () => {
    expect(() => updateFavicon('rose')).not.toThrow()
  })
})
