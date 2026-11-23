// @vitest-environment node
import { readFile } from 'node:fs/promises'
import { join } from 'node:path'

import { describe, expect, it } from 'vitest'

import {
  maximumCachedTiles,
  navigationDenylist,
  precachePatterns,
  preconnectTags,
  pwaOptions,
  themeColor,
  tilePattern,
} from '../pwa.config'
import { readConfig } from '../src/config/env'

const config = readConfig({})
const tiles = [config.mapTilesUrl, config.satelliteTilesUrl]
const options = pwaOptions(tiles)
const runtime = options.workbox?.runtimeCaching ?? []
const pattern = tilePattern(tiles)

const neverCached = [
  `${config.gatewayUrl}/api/v1/plots`,
  `${config.gatewayUrl}/api/v1/plots/preview`,
  `${config.gatewayUrl}/api/v1/users/profile`,
  `${config.oidcAuthority}/protocol/openid-connect/token`,
  `${config.oidcAuthority}/.well-known/openid-configuration`,
  `${config.geocodingUrl}/search?q=Київ`,
  'http://localhost:4173/api/v1/plots',
]

describe('service worker configuration', () => {
  it('caches only map tiles at runtime, at most 200 of them', () => {
    expect(runtime).toHaveLength(1)
    expect(runtime[0]).toMatchObject({
      urlPattern: pattern,
      handler: 'CacheFirst',
      options: { expiration: { maxEntries: maximumCachedTiles } },
    })
    expect(maximumCachedTiles).toBe(200)
    expect(pattern.test('https://tile.openstreetmap.org/15/19168/11083.png')).toBe(true)
    expect(
      pattern.test('https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/15/11083/19168'),
    ).toBe(true)
  })

  it.each(neverCached)('never caches %s', (url) => {
    expect(pattern.test(url)).toBe(false)
  })

  it('does not answer the sign-in callback or the API from the app shell', () => {
    const denied = (path: string) => navigationDenylist.some((rule) => rule.test(path))

    expect(options.workbox?.navigateFallback).toBe('/index.html')
    expect(options.workbox?.navigateFallbackDenylist).toBe(navigationDenylist)
    expect(denied('/auth/callback')).toBe(true)
    expect(denied('/api/v1/plots')).toBe(true)
    expect(denied('/plots/1')).toBe(false)
  })

  it('precaches the shell and only the latin and cyrillic woff2 fonts', () => {
    expect(precachePatterns.filter((glob) => glob.includes('montserrat'))).toEqual([
      'assets/montserrat-latin-[0-9]*.woff2',
      'assets/montserrat-cyrillic-[0-9]*.woff2',
    ])
    expect(precachePatterns.join(' ')).not.toContain('woff,')
  })

  it('uses the primary token as the theme colour and has every icon', async () => {
    const tokens = await readFile(join(import.meta.dirname, '../src/ui/styles/tokens.css'), 'utf8')
    const primary = /--color-primary:\s*(#[0-9A-Fa-f]{6});/.exec(tokens)?.[1]
    const manifest = options.manifest

    expect(manifest && manifest.theme_color).toBe(primary)
    expect(themeColor).toBe(primary)
    expect(manifest && manifest.display).toBe('standalone')
    expect(manifest && manifest.icons?.map((icon) => `${icon.sizes} ${icon.purpose}`)).toEqual([
      '192x192 any',
      '512x512 any',
      '512x512 maskable',
    ])
  })

  it('preconnects to the tile servers and registers the worker without blocking the page', () => {
    expect(preconnectTags(tiles)).toContain('<link rel="preconnect" href="https://tile.openstreetmap.org" />')
    expect(preconnectTags(tiles)).toContain('<link rel="preconnect" href="https://server.arcgisonline.com" />')
    expect(options.injectRegister).toBe('script-defer')
  })
})
