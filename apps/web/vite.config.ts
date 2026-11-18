import vue from '@vitejs/plugin-vue'
import { loadEnv } from 'vite'
import { VitePWA } from 'vite-plugin-pwa'
import { defineConfig } from 'vitest/config'

import { preconnectTags, pwaOptions } from './pwa.config'
import { readConfig } from './src/config/env'

const startPage = '/src/ui/views/CalcView.vue'

export default defineConfig(({ mode }) => {
  const config = readConfig(loadEnv(mode, process.cwd(), 'VITE_'))
  const tiles = [config.mapTilesUrl, config.satelliteTilesUrl]

  return {
    plugins: [
      vue(),
      VitePWA(pwaOptions(tiles)),
      {
        name: 'arqetra-start-page-hints',
        transformIndexHtml: (html, context) => {
          const startChunks = Object.values(context.bundle ?? {})
            .filter((chunk) => chunk.type === 'chunk' && chunk.facadeModuleId?.endsWith(startPage) === true)
            .flatMap((chunk) => (chunk.type === 'chunk' ? [chunk.fileName, ...chunk.imports] : []))
          const entries = Object.values(context.bundle ?? {}).flatMap((chunk) =>
            chunk.type === 'chunk' && chunk.isEntry ? [chunk.fileName] : [],
          )
          const preloads = [...new Set(startChunks)]
            .filter((file) => !entries.includes(file) && !html.includes(`/${file}`))
            .map((file) => `<link rel="modulepreload" crossorigin href="/${file}" />`)

          return html.replace('</head>', `  ${[preconnectTags(tiles), ...preloads].join('\n    ')}\n  </head>`)
        },
      },
    ],
    server: {
      port: 5173,
      strictPort: true,
    },
    preview: {
      port: 4173,
      strictPort: true,
    },
    test: {
      environment: 'jsdom',
      include: ['src/**/*.test.ts', 'tests/**/*.test.ts'],
    },
  }
})
