// @vitest-environment node
import { execFile } from 'node:child_process'
import { rm, writeFile } from 'node:fs/promises'
import { join } from 'node:path'
import { promisify } from 'node:util'

import { afterAll, describe, expect, it } from 'vitest'

const run = promisify(execFile)
const appRoot = join(import.meta.dirname, '..')
const rawProbe = join('src', 'ui', 'components', 'boundaryProbeRawText.vue')
const translatedProbe = join('src', 'ui', 'components', 'boundaryProbeTranslated.vue')
const lintTimeout = 120_000

afterAll(async () => {
  await Promise.all([rawProbe, translatedProbe].map((file) => rm(join(appRoot, file), { force: true })))
})

async function lint(file: string, content: string): Promise<{ code: number; stdout: string }> {
  await writeFile(join(appRoot, file), content)
  try {
    const { stdout } = await run('npx', ['eslint', file], { cwd: appRoot })

    return { code: 0, stdout }
  } catch (error) {
    return error as { code: number; stdout: string }
  }
}

describe('untranslated text in templates', () => {
  it(
    'fails the lint for raw text and a raw aria-label',
    async () => {
      const result = await lint(
        rawProbe,
        '<template>\n  <button\n    type="button"\n    aria-label="Закрити"\n  >\n    Зберегти\n  </button>\n</template>\n',
      )

      expect(result.code).toBe(1)
      expect(result.stdout.match(/vue\/no-bare-strings-in-template/g)).toHaveLength(2)
    },
    lintTimeout,
  )

  it(
    'accepts translated text',
    async () => {
      const result = await lint(
        translatedProbe,
        '<template>\n  <button\n    type="button"\n    :aria-label="$t(\'app.cancel\')"\n  >\n    {{ $t(\'app.signIn\') }}\n  </button>\n</template>\n',
      )

      expect(result.code).toBe(0)
    },
    lintTimeout,
  )
})
