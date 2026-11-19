// @vitest-environment node
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

import { describe, expect, it } from 'vitest'

const styles = join(import.meta.dirname, '..', 'src', 'ui', 'styles')
const tokens = readFileSync(join(styles, 'tokens.css'), 'utf8')
const motion = readFileSync(join(styles, 'motion.css'), 'utf8')
const interaction = readFileSync(join(styles, 'interaction.css'), 'utf8')
const plotMap = readFileSync(join(import.meta.dirname, '..', 'src', 'ui', 'components', 'PlotMap.vue'), 'utf8')
const keycloakTheme = readFileSync(
  join(import.meta.dirname, '..', '..', '..', 'infra', 'keycloak', 'themes', 'arqetra', 'login', 'resources', 'css', 'arqetra.css'),
  'utf8',
)
const allowedTransitions = ['background-color', 'border-color', 'box-shadow', 'opacity', 'transform']

function transitionedProperties(css: string): string[] {
  return [...css.matchAll(/transition:\s*([^;]+);/gu)].flatMap((match) =>
    (match[1] ?? '').split(',').map((part) => part.trim().split(/\s+/u)[0] ?? ''),
  )
}

function hoverRulesOutsideMedia(css: string): string[] {
  const outside = css.replace(/@media \((?:hover: hover|prefers-reduced-motion: reduce)\) \{[\s\S]*?\n\}/gu, '')

  return outside.split('\n').filter((line) => line.includes(':hover'))
}

function reducedBlock(css: string): string {
  const start = css.indexOf('@media (prefers-reduced-motion: reduce)')
  expect(start).toBeGreaterThan(-1)

  return css.slice(start)
}

describe('motion', () => {
  it('defines the motion tokens', () => {
    expect(tokens).toContain('--motion-fast: 150ms;')
    expect(tokens).toContain('--motion-base: 250ms;')
    expect(tokens).toMatch(/--ease-out: cubic-bezier\(/u)
    expect(tokens).toMatch(/--ease-spring: cubic-bezier\(/u)
  })

  it('switches every animation and transition off when motion is reduced', () => {
    expect(reducedBlock(tokens)).toContain('--motion-base: 0ms;')
    const block = reducedBlock(motion)
    expect(block).toContain('animation-duration: 0.01ms !important;')
    expect(block).toContain('transition-duration: 0.01ms !important;')
    expect(block).toContain('::view-transition-group(*)')
  })

  it('animates only transform and opacity, except drawing the outline', () => {
    const animated = [...motion.matchAll(/@keyframes [\w-]+ \{([\s\S]*?)\n\}/gu)].map((match) => match[1] ?? '')
    const properties = new Set(animated.flatMap((body) => [...body.matchAll(/^\s+([\w-]+):/gmu)].map((match) => match[1])))

    expect([...properties].sort()).toEqual(['opacity', 'scale', 'stroke-dashoffset', 'transform'])
  })

  it('keeps the points on the map still', () => {
    const style = plotMap.slice(plotMap.indexOf('<style'))

    expect(style).not.toMatch(/animation:|scale:|translate:/u)
    expect(plotMap).not.toContain('style.translate')
  })

  it('animates hover and press only with the allowed properties and the fast token', () => {
    for (const css of [interaction, keycloakTheme]) {
      const properties = transitionedProperties(css)
      expect(properties.length).toBeGreaterThan(0)
      expect(properties.every((property) => allowedTransitions.includes(property))).toBe(true)
    }
    expect([...interaction.matchAll(/transition:[^;]+;/gu)].every((match) => !/\d+m?s/u.test(match[0]))).toBe(true)
    expect(interaction).toContain('var(--motion-fast)')
  })

  it('shows hover effects only on devices that hover and keeps a focus ring', () => {
    expect(hoverRulesOutsideMedia(interaction)).toEqual([])
    expect(interaction).toMatch(/:focus-visible \{\s+outline: var\(--focus-ring-width\) solid var\(--color-focus-ring\);/u)
    expect(interaction).toContain('.ui-nav-link::after')
    expect(interaction).toMatch(/transform: scaleX\(0\);/u)
  })

  it('changes only colours on hover when motion is reduced', () => {
    const block = reducedBlock(interaction)
    expect(block).toContain('.ui-primary:hover')
    expect(block).toContain('.ui-card:hover')
    expect(block).toContain('transform: none !important;')
    expect(reducedBlock(keycloakTheme)).toContain('transform: none !important;')
  })
})
