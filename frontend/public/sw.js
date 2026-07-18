const CACHE_NAME = 'progress-tracker-shell-__BUILD_ID__'
const BUILD_ASSETS = []
const APP_SHELL = [
  '/',
  '/manifest.webmanifest',
  '/favicon.svg',
  '/icons/icon-192.png',
  '/icons/icon-512.png',
  '/icons/apple-touch-icon.png',
  ...BUILD_ASSETS,
]

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then((cache) => cache.addAll(APP_SHELL))
      .then(() => self.skipWaiting()),
  )
})

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys()
      .then((names) => Promise.all(
        names.filter((name) => name !== CACHE_NAME).map((name) => caches.delete(name)),
      ))
      .then(() => self.clients.claim()),
  )
})

self.addEventListener('fetch', (event) => {
  const request = event.request
  const url = new URL(request.url)

  if (request.method !== 'GET' || url.origin !== self.location.origin || url.pathname.startsWith('/api/')) {
    return
  }

  if (request.mode === 'navigate') {
    event.respondWith(networkFirstNavigation(request))
    return
  }

  event.respondWith(cacheFirstStatic(request))
})

self.addEventListener('notificationclick', (event) => {
  event.notification.close()
  const path = typeof event.notification.data?.url === 'string' && event.notification.data.url.startsWith('/')
    ? event.notification.data.url
    : '/'
  const targetURL = new URL(path, self.location.origin).href
  event.waitUntil(
    self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clients) => {
      const existingClient = clients.find((client) => new URL(client.url).origin === self.location.origin)
      if (existingClient) {
        return existingClient.navigate(targetURL).then(() => existingClient.focus())
      }
      return self.clients.openWindow(targetURL)
    }),
  )
})

self.addEventListener('push', (event) => {
  let payload = {}
  try {
    payload = event.data?.json() || {}
  } catch {
    payload = { body: event.data?.text() || '' }
  }

  event.waitUntil(self.registration.showNotification(payload.title || 'Progress Tracker', {
    body: payload.body || '',
    tag: payload.tag || 'progress-tracker',
    icon: '/icons/icon-192.png',
    badge: '/icons/icon-192.png',
    data: { url: typeof payload.url === 'string' ? payload.url : '/' },
  }))
})

async function networkFirstNavigation(request) {
  const cache = await caches.open(CACHE_NAME)
  try {
    const response = await fetch(request)
    if (response.ok) {
      await cache.put('/', response.clone())
    }
    return response
  } catch {
    return (await cache.match('/')) || Response.error()
  }
}

async function cacheFirstStatic(request) {
  const cached = await caches.match(request, { ignoreVary: true })
  if (cached) {
    return cached
  }

  const response = await fetch(request)
  if (response.ok && response.type === 'basic') {
    const cache = await caches.open(CACHE_NAME)
    await cache.put(request, response.clone())
  }
  return response
}
