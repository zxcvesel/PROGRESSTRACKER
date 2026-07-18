import { expect, test } from '@playwright/test'

test('installs the production service worker and reloads the app shell offline', async ({ context, page }) => {
  await page.goto('/')
  await page.waitForFunction(async () => {
    if (!('serviceWorker' in navigator)) return false
    await navigator.serviceWorker.ready
    return Boolean(navigator.serviceWorker.controller)
  })

  const manifestResponse = await page.request.get('/manifest.webmanifest')
  expect(manifestResponse.ok()).toBe(true)
  const manifest = await manifestResponse.json()
  expect(manifest).toMatchObject({
    display: 'standalone',
    start_url: '/',
    scope: '/',
  })
  expect(manifest.icons).toEqual(expect.arrayContaining([
    expect.objectContaining({ sizes: '192x192', type: 'image/png' }),
    expect.objectContaining({ sizes: '512x512', purpose: 'any maskable' }),
  ]))

  const workerResponse = await page.request.get('/sw.js')
  expect(workerResponse.ok()).toBe(true)
  const workerSource = await workerResponse.text()
  expect(workerSource).toContain("url.pathname.startsWith('/api/')")
  expect(workerSource).toContain("addEventListener('push'")

  await context.setOffline(true)
  await page.reload()
  await expect(page.getByRole('heading', { name: 'Welcome' })).toBeVisible()
  await expect(page.getByRole('status')).toContainText('Offline')
  await context.setOffline(false)
})

test('keeps the primary interface inside a mobile viewport', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Welcome' })).toBeVisible()

  const dimensions = await page.evaluate(() => ({
    viewportWidth: document.documentElement.clientWidth,
    contentWidth: document.documentElement.scrollWidth,
  }))
  expect(dimensions.contentWidth).toBeLessThanOrEqual(dimensions.viewportWidth)
  await expect(page.getByRole('button', { name: 'Sign in', exact: true })).toBeInViewport()
})
