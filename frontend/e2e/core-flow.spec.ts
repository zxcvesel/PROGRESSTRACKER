import { expect, test } from '@playwright/test'

test('registers, creates a goal, and saves a server-timed session', async ({ page }) => {
  const email = `e2e-${Date.now()}@example.com`

  await page.goto('/')
  await page.getByRole('button', { name: /No account yet.*Create account/i }).click()
  await page.getByLabel('Email').fill(email)
  await page.getByLabel('Name').fill('E2E User')
  await page.getByLabel('Password', { exact: true }).fill('Password123!')
  await page.getByLabel('Confirm password').fill('Password123!')
  await page.getByRole('button', { name: 'Create account', exact: true }).click()

  await expect(page.getByText('Create your first goal')).toBeVisible()
  const pushKeyResponse = await page.request.get('/api/push/public-key')
  expect(pushKeyResponse.ok()).toBe(true)
  expect(await pushKeyResponse.json()).toEqual({
    publicKey: expect.stringMatching(/^[A-Za-z0-9_-]+$/),
  })
  await page.locator('.empty-state').getByRole('button', { name: 'Create goal' }).click()
  await page.getByLabel('Title').fill('Playwright goal')
  await page.getByLabel('Description').fill('Verify the complete user flow')
  await page.getByLabel('Days').fill('30')
  await page.getByLabel('Daily target hours').fill('0')
  await page.getByLabel('Minutes').fill('5')
  await page.locator('form.entry-form').getByRole('button', { name: 'Create goal' }).click()

  await expect(page.getByRole('heading', { name: 'Playwright goal' })).toBeVisible()
  await page.getByRole('button', { name: 'Start session' }).click()
  await expect(page.getByText('Session running')).toBeVisible()
  await page.getByRole('button', { name: 'Pause' }).click()
  await page.getByRole('button', { name: 'Finish session' }).click()
  await expect(page.getByRole('heading', { name: 'Session completed' })).toBeVisible()
  await page.getByLabel('Notes').fill('Completed through the server timer')
  await page.getByRole('button', { name: 'Save session' }).click()

  await expect(page.getByText('1m / 5m')).toBeVisible()

  await page.getByRole('button', { name: 'Back to goals' }).click()
  await page.getByRole('button', { name: 'Open settings' }).click()
  const settingsDrawer = page.locator('aside.settings-drawer')
  await expect(settingsDrawer).toBeVisible()
  await expect(settingsDrawer.locator(':scope > details.settings-group')).toHaveCount(7)

  await settingsDrawer.getByRole('heading', { name: 'Account' }).click()
  await expect(settingsDrawer.getByRole('button', { name: 'Log out', exact: true })).toHaveCount(1)
  await settingsDrawer.locator('details.settings-details > summary').filter({ hasText: 'Display name' }).click()
  await expect(settingsDrawer.getByRole('button', { name: 'Save', exact: true })).toBeVisible()

  await settingsDrawer.getByRole('button', { name: 'Close settings' }).click()
  await page.getByRole('button', { name: 'Stats' }).click()
  const completionRing = page.locator('.stats-completion-ring')
  await expect(completionRing).toBeVisible()
  await expect(completionRing).toHaveCSS('background-image', /conic-gradient/)
  const ringCoreBackground = await completionRing.evaluate((element) => (
    window.getComputedStyle(element, '::before').backgroundColor
  ))
  expect(ringCoreBackground).not.toBe('rgba(0, 0, 0, 0)')
})

test('resets a forgotten password and invalidates the old one', async ({ page }) => {
  const email = `reset-${Date.now()}@example.com`

  await page.goto('/')
  await page.getByRole('button', { name: /No account yet.*Create account/i }).click()
  await page.getByLabel('Email').fill(email)
  await page.getByLabel('Password', { exact: true }).fill('Password123!')
  await page.getByLabel('Confirm password').fill('Password123!')
  await page.getByRole('button', { name: 'Create account', exact: true }).click()
  await expect(page.getByText('Create your first goal')).toBeVisible()

  await page.request.post('/api/auth/logout', {
    headers: { Origin: 'http://127.0.0.1:4174' },
  })
  await page.reload()
  await page.getByRole('button', { name: 'Forgot password?' }).click()
  await page.getByLabel('Email').fill(email)
  await page.getByRole('button', { name: 'Send reset link' }).click()
  await expect(page.getByRole('heading', { name: 'Set a new password' })).toBeVisible()
  await page.getByLabel('Password', { exact: true }).fill('NewPassword123!')
  await page.getByLabel('Confirm password').fill('NewPassword123!')
  await page.getByRole('button', { name: 'Save new password' }).click()
  await expect(page.getByText('Password updated. Sign in with your new password.')).toBeVisible()

  await page.getByLabel('Email').fill(email)
  await page.getByLabel('Password', { exact: true }).fill('NewPassword123!')
  await page.getByRole('button', { name: 'Sign in', exact: true }).click()
  await expect(page.getByText('Create your first goal')).toBeVisible()
})
