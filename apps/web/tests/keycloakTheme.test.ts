// @vitest-environment node
import { readFile } from 'node:fs/promises'
import { join } from 'node:path'

import { describe, expect, it } from 'vitest'

const repository = join(import.meta.dirname, '../../..')
const theme = join(repository, 'infra/keycloak/themes/arqetra/login')

async function keys(file: string): Promise<string[]> {
  return (await readFile(join(theme, 'messages', file), 'utf8'))
    .split('\n')
    .filter((line) => line.includes('=') && !line.startsWith('#'))
    .map((line) => line.slice(0, line.indexOf('=')).trim())
}

describe('Keycloak login theme', () => {
  it('uses the same design tokens as the web app', async () => {
    const appTokens = await readFile(join(repository, 'apps/web/src/ui/styles/tokens.css'), 'utf8')
    const themeTokens = await readFile(join(theme, 'resources/css/tokens.css'), 'utf8')

    expect(themeTokens).toBe(appTokens)
  })

  it('has a Ukrainian text for every English one', async () => {
    const ukrainian = new Set(await keys('messages_uk.properties'))

    expect((await keys('messages_en.properties')).filter((key) => !ukrainian.has(key))).toEqual([])
  })

  it('takes every colour in its stylesheet from the tokens', async () => {
    const css = await readFile(join(theme, 'resources/css/arqetra.css'), 'utf8')

    expect(css.match(/#[0-9a-fA-F]{3,8}\b|rgba?\(/g)).toBeNull()
  })
})
