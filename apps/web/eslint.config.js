import { defineConfigWithVueTs, vueTsConfigs } from '@vue/eslint-config-typescript'
import boundaries from 'eslint-plugin-boundaries'
import pluginVue from 'eslint-plugin-vue'

const layers = ['domain', 'port', 'service', 'client', 'auth', 'repository', 'config', 'ui', 'bootstrap']

const allow = (from, to) => ({
  from: { element: { types: [from] } },
  allow: { to: { element: { types: to } } },
})

export default defineConfigWithVueTs(
  {
    ignores: ['dist/**', 'dev-dist/**', 'coverage/**'],
  },
  pluginVue.configs['flat/recommended'],
  vueTsConfigs.recommended,
  {
    files: ['src/**/*.{ts,vue}'],
    plugins: { boundaries },
    settings: {
      'import/resolver': {
        node: { extensions: ['.ts', '.vue', '.js'] },
      },
      'boundaries/elements': layers.map((layer) => ({ type: layer, pattern: `src/${layer}` })),
    },
    rules: {
      'boundaries/dependencies': [
        'error',
        {
          default: 'disallow',
          policies: [
            allow('port', ['domain']),
            allow('service', ['port', 'domain']),
            allow('client', ['port', 'domain', 'config']),
            allow('auth', ['port', 'domain', 'config']),
            allow('repository', ['port', 'domain', 'config']),
            allow('ui', ['service', 'domain']),
            allow('bootstrap', layers),
          ],
          message: '{{ from.element.types.[0] }} must not depend on {{ to.element.types.[0] }}',
        },
      ],
      'boundaries/no-unknown-files': 'error',
      'boundaries/no-unknown-dependencies': 'error',
    },
  },
  {
    files: ['src/**/*.vue'],
    rules: {
      'vue/no-bare-strings-in-template': [
        'error',
        { allowlist: ['Arqetra', '·', '—', '−', '+', '←', '«', '»', '(', ')', ',', '.', ':', '*', '№', '%', '/'] },
      ],
    },
  },
  {
    files: ['src/domain/**/*.ts', 'src/port/**/*.ts', 'src/service/**/*.ts'],
    ignores: ['**/*.test.ts'],
    rules: {
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            {
              regex: '^[^.]',
              message: 'domain, port and service use only relative imports: no Vue, HTTP or browser libraries',
            },
          ],
        },
      ],
    },
  },
)
