import { afterEach, describe, expect, it, vi } from 'vitest'
import { ensurePushSubscription, removePushSubscription } from './pwa'

describe('PWA push subscriptions', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    Reflect.deleteProperty(navigator, 'serviceWorker')
  })

  it('sends an existing browser subscription to the authenticated API', async () => {
    const payload = {
      endpoint: 'https://web.push.apple.com/example',
      keys: { auth: 'auth-key', p256dh: 'public-key' },
    }
    const registration = {
      pushManager: {
        getSubscription: vi.fn().mockResolvedValue({ toJSON: () => payload }),
      },
    }
    setPushEnvironment(registration)
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }))
    vi.stubGlobal('fetch', fetchMock)

    await expect(ensurePushSubscription()).resolves.toBe(true)
    expect(fetchMock).toHaveBeenCalledWith('/api/push/subscriptions', expect.objectContaining({
      method: 'POST',
      credentials: 'same-origin',
      body: JSON.stringify(payload),
    }))
  })

  it('creates a browser subscription with the server VAPID key', async () => {
    const payload = {
      endpoint: 'https://web.push.apple.com/new-subscription',
      keys: { auth: 'auth-key', p256dh: 'public-key' },
    }
    const subscribe = vi.fn().mockResolvedValue({ toJSON: () => payload })
    const registration = {
      pushManager: {
        getSubscription: vi.fn().mockResolvedValue(null),
        subscribe,
      },
    }
    setPushEnvironment(registration)
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ publicKey: 'AQIDBA' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }))
      .mockResolvedValueOnce(new Response(null, { status: 204 }))
    vi.stubGlobal('fetch', fetchMock)

    await expect(ensurePushSubscription()).resolves.toBe(true)
    expect(subscribe).toHaveBeenCalledWith({
      userVisibleOnly: true,
      applicationServerKey: new Uint8Array([1, 2, 3, 4]),
    })
    expect(fetchMock).toHaveBeenNthCalledWith(1, '/api/push/public-key', {
      credentials: 'same-origin',
    })
  })

  it('removes the subscription from the API and browser', async () => {
    const unsubscribe = vi.fn().mockResolvedValue(true)
    const registration = {
      pushManager: {
        getSubscription: vi.fn().mockResolvedValue({
          endpoint: 'https://fcm.googleapis.com/example',
          unsubscribe,
        }),
      },
    }
    setPushEnvironment(registration)
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }))
    vi.stubGlobal('fetch', fetchMock)

    await removePushSubscription()

    expect(fetchMock).toHaveBeenCalledWith('/api/push/subscriptions', expect.objectContaining({
      method: 'DELETE',
      body: JSON.stringify({ endpoint: 'https://fcm.googleapis.com/example' }),
    }))
    expect(unsubscribe).toHaveBeenCalledOnce()
  })
})

function setPushEnvironment(registration: object) {
  vi.stubGlobal('PushManager', class PushManager {})
  Object.defineProperty(navigator, 'serviceWorker', {
    configurable: true,
    value: {
      ready: Promise.resolve(registration),
      getRegistration: vi.fn().mockResolvedValue(registration),
    },
  })
}
