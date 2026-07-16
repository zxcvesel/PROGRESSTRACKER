import { mkdirSync } from 'node:fs'
import { resolve } from 'node:path'
import { spawnSync } from 'node:child_process'

const cacheDirectory = resolve('../.cache')
const goCache = resolve(cacheDirectory, 'go-build')
const executable = resolve(cacheDirectory, process.platform === 'win32' ? 'e2e-api.exe' : 'e2e-api')
mkdirSync(cacheDirectory, { recursive: true })

const result = spawnSync('go', ['build', '-buildvcs=false', '-o', executable, './cmd/api'], {
  cwd: resolve('../backend'),
  env: { ...process.env, GOCACHE: goCache },
  stdio: 'inherit',
})

if (result.status !== 0) {
  process.exit(result.status ?? 1)
}
