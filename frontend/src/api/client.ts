export type ApiErrorResponse = {
  error?: string
}

export async function requestAPI(path: string, options: RequestInit = {}) {
  const headers = new Headers(options.headers)
  return fetch(path, { ...options, credentials: 'same-origin', headers })
}

export async function readAPIError(response: Response, fallback: string) {
  try {
    const data = (await response.clone().json()) as ApiErrorResponse
    if (typeof data.error === 'string' && data.error.trim() !== '') {
      return `${fallback}: ${data.error}`
    }
  } catch {
    // Network and proxy errors may not include a JSON response body.
  }

  return fallback
}
