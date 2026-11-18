// @vitest-environment node
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

import { describe, expect, it } from 'vitest'

const styles = join(import.meta.dirname, '..', 'src', 'ui', 'styles')
const tokens = readFileSync(join(styles, 'tokens.css'), 'utf8')
const motion = readFileSync(join(styles, 'motion.css'), 'utf8')

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
})
