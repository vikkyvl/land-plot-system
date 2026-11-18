import type { VitePWAOptions } from 'vite-plugin-pwa'

export const themeColor = '#2F6B4F'
export const backgroundColor = '#F6F5F0'
export const tileCacheName = 'map-tiles'
export const maximumCachedTiles = 200
const tileCacheDays = 30
const secondsPerDay = 86_400
const opaqueStatus = 0
const okStatus = 200

export const precachePatterns = [
  '**/*.{js,css,html,svg,png,webmanifest}',
  'assets/montserrat-latin-[0-9]*.woff2',
  'assets/montserrat-cyrillic-[0-9]*.woff2',
]

export const navigationDenylist = [/^\/auth\//, /^\/api\//]

export function tileOrigins(templates: readonly string[]): string[] {
  return [...new Set(templates.map((template) => new URL(template.replace(/[{}]/g, '')).origin))]
}

function escapeForPattern(text: string): string {
  return text.replace(/[.*+?^${}()|[\]\\/]/g, '\\$&')
}

export function tilePattern(templates: readonly string[]): RegExp {
  return new RegExp(`^(?:${tileOrigins(templates).map(escapeForPattern).join('|')})/`)
}

export function preconnectTags(templates: readonly string[]): string {
  return tileOrigins(templates)
    .map((origin) => `<link rel="preconnect" href="${origin}" />`)
    .join('\n    ')
}

export function pwaOptions(tileTemplates: readonly string[]): Partial<VitePWAOptions> {
  return {
    registerType: 'autoUpdate',
    injectRegister: 'script-defer',
    manifest: {
      name: 'Arqetra — площа земельної ділянки',
      short_name: 'Arqetra',
      description: 'Розрахунок площі земельної ділянки за точками на карті',
      lang: 'uk',
      start_url: '/',
      scope: '/',
      display: 'standalone',
      orientation: 'any',
      theme_color: themeColor,
      background_color: backgroundColor,
      icons: [
        { src: '/icons/icon-192.png', sizes: '192x192', type: 'image/png', purpose: 'any' },
        { src: '/icons/icon-512.png', sizes: '512x512', type: 'image/png', purpose: 'any' },
        { src: '/icons/icon-maskable-512.png', sizes: '512x512', type: 'image/png', purpose: 'maskable' },
      ],
    },
    workbox: {
      globPatterns: precachePatterns,
      navigateFallback: '/index.html',
      navigateFallbackDenylist: navigationDenylist,
      cleanupOutdatedCaches: true,
      runtimeCaching: [
        {
          urlPattern: tilePattern(tileTemplates),
          handler: 'CacheFirst',
          options: {
            cacheName: tileCacheName,
            expiration: { maxEntries: maximumCachedTiles, maxAgeSeconds: tileCacheDays * secondsPerDay },
            cacheableResponse: { statuses: [opaqueStatus, okStatus] },
          },
        },
      ],
    },
  }
}
