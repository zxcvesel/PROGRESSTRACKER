import { afterEach, describe, expect, it, vi } from 'vitest'
import { readAPIError, requestAPI } from './client'

describe('API client', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('always sends same-origin credentials', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response('{}', { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    await requestAPI('/api/goals', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
    })

    expect(fetchMock).toHaveBeenCalledOnce()
    expect(fetchMock).toHaveBeenCalledWith('/api/goals', expect.objectContaining({
      method: 'POST',
      credentials: 'same-origin',
    }))
  })

  it('returns a structured API error when available', async () => {
    const response = new Response(JSON.stringify({ error: 'daily target is already completed' }), {
      status: 409,
      headers: { 'Content-Type': 'application/json' },
    })

    await expect(readAPIError(response, 'Could not start')).resolves.toBe(
      'Could not start: daily target is already completed',
    )
  })

  it('uses the fallback for non-JSON failures', async () => {
    const response = new Response('Bad gateway', { status: 502 })
    await expect(readAPIError(response, 'Connection failed')).resolves.toBe('Connection failed')
  })
})
