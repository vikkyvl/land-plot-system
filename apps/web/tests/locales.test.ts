// @vitest-environment node
import { describe, expect, it } from 'vitest'

import en from '../src/ui/locales/en.json'
import uk from '../src/ui/locales/uk.json'

type Messages = { [key: string]: string | Messages }

function flatten(messages: Messages, prefix = ''): Map<string, string> {
  const entries = new Map<string, string>()
  for (const [key, value] of Object.entries(messages)) {
    if (typeof value === 'string') {
      entries.set(prefix + key, value)
    } else {
      for (const [nested, text] of flatten(value, `${prefix}${key}.`)) {
        entries.set(nested, text)
      }
    }
  }

  return entries
}

const ukrainian = flatten(uk)
const english = flatten(en)
const parameters = (text: string) => [...text.matchAll(/\{(\w+)\}/g)].map((match) => match[1]).sort()

describe('locale files', () => {
  it('have the same keys', () => {
    expect([...ukrainian.keys()].filter((key) => !english.has(key))).toEqual([])
    expect([...english.keys()].filter((key) => !ukrainian.has(key))).toEqual([])
  })

  it('use the same parameters in both languages', () => {
    for (const [key, text] of ukrainian) {
      expect([...new Set(parameters(english.get(key) ?? ''))], key).toEqual([...new Set(parameters(text))])
    }
  })

  it('have no empty texts', () => {
    expect([...ukrainian, ...english].filter(([, text]) => text.trim() === '')).toEqual([])
  })

  it('give Ukrainian counts three plural forms and English two', () => {
    for (const key of ['counts.points', 'counts.plots', 'counts.versions']) {
      expect(ukrainian.get(key)?.split('|'), key).toHaveLength(3)
      expect(english.get(key)?.split('|'), key).toHaveLength(2)
    }
  })
})
