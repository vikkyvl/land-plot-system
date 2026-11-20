// @vitest-environment node
import { execFile } from 'node:child_process'
import { mkdir, rm, writeFile } from 'node:fs/promises'
import { join } from 'node:path'
import { promisify } from 'node:util'

import { afterAll, beforeAll, describe, expect, it } from 'vitest'

const run = promisify(execFile)
const appRoot = join(import.meta.dirname, '..')
const probeTarget = 'boundaryProbeTarget'
const lintTimeout = 120_000

type Dependency = { from: string; to: string; allowed: boolean }

const dependencies: Dependency[] = [
  { from: 'ui', to: 'client', allowed: false },
  { from: 'ui', to: 'auth', allowed: false },
  { from: 'ui', to: 'repository', allowed: false },
  { from: 'domain', to: 'port', allowed: false },
  { from: 'port', to: 'service', allowed: false },
  { from: 'service', to: 'client', allowed: false },
  { from: 'client', to: 'ui', allowed: false },
  { from: 'ui', to: 'service', allowed: true },
  { from: 'service', to: 'port', allowed: true },
  { from: 'client', to: 'config', allowed: true },
  { from: 'bootstrap', to: 'client', allowed: true },
]

const sourceFile = ({ from, to }: Dependency) => join('src', from, `boundaryProbeFrom_${to}.ts`)
const targetFile = (layer: string) => join('src', layer, `${probeTarget}.ts`)
const createdFiles = [
  ...new Set([...dependencies.map(sourceFile), ...dependencies.map(({ to }) => targetFile(to))]),
]

type LintResult = { filePath: string; messages: { ruleId: string | null }[] }

async function lint(files: string[]): Promise<{ exitCode: number; output: string }> {
  try {
    const { stdout } = await run('npx', ['eslint', '--format', 'json', ...files], { cwd: appRoot })
    return { exitCode: 0, output: stdout }
  } catch (error) {
    const failed = error as { code: number; stdout: string }
    return { exitCode: failed.code, output: failed.stdout }
  }
}

beforeAll(async () => {
  for (const dependency of dependencies) {
    await mkdir(join(appRoot, 'src', dependency.to), { recursive: true })
    await writeFile(join(appRoot, targetFile(dependency.to)), 'export const probe = 1\n')
    await writeFile(
      join(appRoot, sourceFile(dependency)),
      `import { probe } from '../${dependency.to}/${probeTarget}'\n\nexport const value = probe\n`,
    )
  }
})

afterAll(async () => {
  await Promise.all(createdFiles.map((file) => rm(join(appRoot, file), { force: true })))
})

describe('layer boundaries', () => {
  it(
    'reports exactly the forbidden dependencies',
    async () => {
      const { output } = await lint(dependencies.map(sourceFile))
      const results = JSON.parse(output) as LintResult[]
      const violations = (dependency: Dependency) =>
        results
          .find((result) => result.filePath.endsWith(sourceFile(dependency)))
          ?.messages.filter((message) => message.ruleId === 'boundaries/dependencies') ?? []

      for (const dependency of dependencies) {
        expect(violations(dependency), `${dependency.from} → ${dependency.to}`).toHaveLength(
          dependency.allowed ? 0 : 1,
        )
      }
    },
    lintTimeout,
  )

  it(
    'fails npm run lint when a boundary is broken',
    async () => {
      const lintRun = run('npm', ['run', 'lint'], { cwd: appRoot })

      await expect(lintRun).rejects.toMatchObject({
        code: 1,
        stdout: expect.stringContaining('ui must not depend on client'),
      })
    },
    lintTimeout,
  )
})
