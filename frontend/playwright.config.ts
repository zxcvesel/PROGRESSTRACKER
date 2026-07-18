import { defineConfig, devices } from '@playwright/test'
import { resolve } from 'node:path'

const backendExecutable = resolve('../.cache', process.platform === 'win32' ? 'e2e-api.exe' : 'e2e-api')

export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  retries: process.env.CI ? 2 : 0,
  reporter: process.env.CI ? 'github' : 'list',
  use: {
    baseURL: 'http://127.0.0.1:4174',
    trace: 'on-first-retry',
    ...devices['Desktop Chrome'],
  },
  webServer: process.env.E2E_EXTERNAL_SERVERS ? undefined : [
    {
      command: `"${backendExecutable}"`,
      cwd: '../backend',
      url: 'http://127.0.0.1:18080/health',
      env: {
        PROGRESS_TRACKER_HOST: '127.0.0.1',
        PROGRESS_TRACKER_PORT: '18080',
        PROGRESS_TRACKER_DB_PATH: 'data/e2e-progress.db',
        PROGRESS_TRACKER_ALLOWED_ORIGINS: 'http://127.0.0.1:4174',
        GOCACHE: resolve('../.cache/go-build'),
      },
      reuseExistingServer: !process.env.CI,
      timeout: 120_000,
    },
    {
      command: 'npm run preview -- --host 127.0.0.1 --port 4174',
      url: 'http://127.0.0.1:4174',
      env: {
        VITE_API_TARGET: 'http://127.0.0.1:18080',
      },
      reuseExistingServer: !process.env.CI,
      timeout: 120_000,
    },
  ],
})
