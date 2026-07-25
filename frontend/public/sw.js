const CACHE_NAME = 'progress-tracker-shell-__BUILD_ID__'
const BUILD_ASSETS = []
const APP_SHELL = [
  '/',
  '/manifest.webmanifest',
  '/manifests/manifest-cyan.webmanifest',
  '/manifests/manifest-purple.webmanifest',
  '/manifests/manifest-green.webmanifest',
  '/manifests/manifest-orange.webmanifest',
  '/favicons/icon-cyan.svg',
  '/favicons/icon-purple.svg',
  '/favicons/icon-green.svg',
  '/favicons/icon-orange.svg',
  '/icons/cyan/icon-192.png',
  '/icons/cyan/icon-512.png',
  '/icons/cyan/apple-touch-icon.png',
  '/icons/purple/icon-192.png',
  '/icons/purple/icon-512.png',
  '/icons/purple/apple-touch-icon.png',
  '/icons/green/icon-192.png',
  '/icons/green/icon-512.png',
  '/icons/green/apple-touch-icon.png',
  '/icons/orange/icon-192.png',
  '/icons/orange/icon-512.png',
  '/icons/orange/apple-touch-icon.png',
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

  event.waitUntil(self.registration.showNotification(payload.title || 'Sparx', {
    body: payload.body || '',
    tag: payload.tag || 'progress-tracker',
    icon: '/icons/cyan/icon-192.png',
    badge: '/icons/cyan/icon-192.png',
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
