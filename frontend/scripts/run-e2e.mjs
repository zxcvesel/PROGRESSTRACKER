import { spawn, spawnSync } from 'node:child_process'
import { resolve } from 'node:path'

const backendExecutable = resolve('../.cache', process.platform === 'win32' ? 'e2e-api.exe' : 'e2e-api')
const backend = spawn(backendExecutable, [], {
  cwd: resolve('../backend'),
  env: {
    ...process.env,
    PROGRESS_TRACKER_HOST: '127.0.0.1',
    PROGRESS_TRACKER_PORT: '18080',
    PROGRESS_TRACKER_DB_PATH: 'data/e2e-progress.db',
    PROGRESS_TRACKER_ALLOWED_ORIGINS: 'http://127.0.0.1:4174',
  },
  stdio: 'inherit',
})
const frontend = spawn(process.execPath, [
  resolve('node_modules/vite/bin/vite.js'),
  'preview',
  '--host', '127.0.0.1',
  '--port', '4174',
], {
  cwd: resolve('.'),
  env: { ...process.env, VITE_API_TARGET: 'http://127.0.0.1:18080' },
  stdio: 'inherit',
})

let exitCode = 1
try {
  await Promise.all([
    waitForURL('http://127.0.0.1:18080/health'),
    waitForURL('http://127.0.0.1:4174'),
  ])
  const result = spawnSync(process.execPath, [
    resolve('node_modules/@playwright/test/cli.js'),
    'test',
    ...process.argv.slice(2),
  ], {
    cwd: resolve('.'),
    env: { ...process.env, E2E_EXTERNAL_SERVERS: 'true' },
    stdio: 'inherit',
  })
  exitCode = result.status ?? 1
} finally {
  stop(frontend)
  stop(backend)
}

process.exit(exitCode)

async function waitForURL(url) {
  const deadline = Date.now() + 120_000
  while (Date.now() < deadline) {
    try {
      const response = await fetch(url)
      if (response.ok) return
    } catch {
      // The process is still starting.
    }
    await new Promise((resolveWait) => setTimeout(resolveWait, 250))
  }
  throw new Error(`Timed out waiting for ${url}`)
}

function stop(child) {
  if (child.exitCode === null && !child.killed) {
    child.kill()
  }
}
