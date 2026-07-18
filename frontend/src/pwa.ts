export function registerServiceWorker() {
  if (!import.meta.env.PROD || !('serviceWorker' in navigator)) {
    return
  }

  window.addEventListener('load', () => {
    void navigator.serviceWorker.register('/sw.js', { scope: '/' }).catch(() => {
      // The app remains usable as a regular website when registration fails.
    })
  })
}

export async function ensurePushSubscription() {
  if (!('serviceWorker' in navigator) || !('PushManager' in window)) {
    return false
  }

  const registration = await navigator.serviceWorker.ready
  let subscription = await registration.pushManager.getSubscription()
  if (!subscription) {
    const keyResponse = await fetch('/api/push/public-key', { credentials: 'same-origin' })
    if (!keyResponse.ok) {
      return false
    }
    const { publicKey } = await keyResponse.json() as { publicKey: string }
    subscription = await registration.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: decodeBase64URL(publicKey),
    })
  }

  const response = await fetch('/api/push/subscriptions', {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(subscription.toJSON()),
  })
  return response.ok
}

export async function removePushSubscription() {
  if (!('serviceWorker' in navigator) || !('PushManager' in window)) {
    return
  }

  const registration = await navigator.serviceWorker.getRegistration()
  const subscription = await registration?.pushManager.getSubscription()
  if (!subscription) {
    return
  }

  try {
    await fetch('/api/push/subscriptions', {
      method: 'DELETE',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ endpoint: subscription.endpoint }),
    })
  } finally {
    await subscription.unsubscribe()
  }
}

function decodeBase64URL(value: string) {
  const padding = '='.repeat((4 - value.length % 4) % 4)
  const binary = window.atob((value + padding).replace(/-/g, '+').replace(/_/g, '/'))
  return Uint8Array.from(binary, (character) => character.charCodeAt(0))
}
