// @vitest-environment node
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative } from 'node:path'

import { describe, expect, it } from 'vitest'

const source = join(import.meta.dirname, '..', 'src')
const geometry =
  /Math\.(?:cos|sin|tan|acos|asin|atan2?|hypot|sqrt|PI)\b|haversine|shoelace|latLngToLayerPoint|\.distanceTo\(|\.project\(/u

function sourceFiles(directory: string): string[] {
  return readdirSync(directory).flatMap((entry) => {
    const path = join(directory, entry)
    if (statSync(path).isDirectory()) {
      return sourceFiles(path)
    }

    return /\.(?:ts|vue)$/u.test(entry) && !entry.includes('.test.') ? [path] : []
  })
}

describe('front-end and back-end boundary', () => {
  it('keeps measurements, intersections and projections out of the web app', () => {
    const offenders = sourceFiles(source)
      .filter((path) => geometry.test(readFileSync(path, 'utf8')))
      .map((path) => relative(source, path))

    expect(offenders).toEqual([])
  })
})
