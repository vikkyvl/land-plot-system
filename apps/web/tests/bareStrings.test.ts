// @vitest-environment node
import { spawnSync } from 'node:child_process'
import { join } from 'node:path'

import { describe, expect, it } from 'vitest'

const appRoot = join(import.meta.dirname, '..')
const probeName = join('src', 'ui', 'components', 'BareStringsProbe.vue')
const lintTimeout = 120_000

function lint(content: string): { code: number | null; stdout: string } {
  const result = spawnSync('npx', ['eslint', '--stdin', '--stdin-filename', probeName], {
    cwd: appRoot,
    input: content,
    encoding: 'utf8',
  })

  return { code: result.status, stdout: result.stdout }
}

describe('untranslated text in templates', () => {
  it(
    'fails the lint for raw text and a raw aria-label',
    () => {
      const result = lint(
        '<template>\n  <button\n    type="button"\n    aria-label="Закрити"\n  >\n    Зберегти\n  </button>\n</template>\n',
      )

      expect(result.code).toBe(1)
      expect(result.stdout.match(/vue\/no-bare-strings-in-template/g)).toHaveLength(2)
    },
    lintTimeout,
  )

  it(
    'accepts translated text',
    () => {
      const result = lint(
        '<template>\n  <button\n    type="button"\n    :aria-label="$t(\'app.cancel\')"\n  >\n    {{ $t(\'app.signIn\') }}\n  </button>\n</template>\n',
      )

      expect(result.code).toBe(0)
    },
    lintTimeout,
  )
})
