// @vitest-environment node
import { existsSync, readFileSync } from 'node:fs'
import { join } from 'node:path'

import { describe, expect, it } from 'vitest'

import en from '../src/ui/locales/en.json'
import uk from '../src/ui/locales/uk.json'

const root = join(import.meta.dirname, '..', '..', '..')
const theme = join(root, 'infra', 'keycloak', 'themes', 'arqetra', 'login')
const read = (...parts: string[]) => readFileSync(join(...parts), 'utf8')

describe('web login and Keycloak theme', () => {
  it('take the Google "G" from one file', () => {
    const logo = read(root, 'apps', 'web', 'public', 'brand', 'google.svg')

    expect(logo).toContain('<svg')
    expect(existsSync(join(theme, 'resources', 'img', 'social', 'google.svg'))).toBe(false)
    expect(read(root, 'docker-compose.yml')).toContain(
      './apps/web/public/brand:/opt/keycloak/themes/arqetra/login/resources/img/social:ro',
    )
    expect(read(root, 'apps', 'web', 'src', 'ui', 'views', 'LoginView.vue')).toContain('brand/google.svg')
  })

  it('share the card size and look through tokens', () => {
    const login = read(root, 'apps', 'web', 'src', 'ui', 'views', 'LoginView.vue')
    const css = read(theme, 'resources', 'css', 'arqetra.css')

    for (const token of ['--login-card-width', '--login-card-radius', '--login-card-shadow', '--login-card-padding']) {
      expect(login).toContain(`var(${token})`)
      expect(css).toContain(`var(${token})`)
    }
  })

  it('name the pages the same way and keep the account subtitle only in the app', () => {
    expect(JSON.stringify(uk.login)).not.toContain('Обліковий запис')
    expect(JSON.stringify(en.login)).not.toContain('"Account"')
    expect(read(theme, 'messages', 'messages_uk.properties')).toContain('loginAccountTitle=Вхід до Arqetra')
    expect(read(theme, 'messages', 'messages_uk.properties')).toContain('registerTitle=Реєстрація в Arqetra')
    expect(read(theme, 'messages', 'messages_en.properties')).toContain('loginAccountTitle=Sign in to Arqetra')
    expect(read(theme, 'messages', 'messages_en.properties')).toContain('registerTitle=Create your Arqetra account')
    expect(read(theme, 'arqetra.ftl')).not.toContain('arqetraIntro')
    expect(read(theme, 'arqetra.ftl')).toContain('arqetra-locale')
  })
})
