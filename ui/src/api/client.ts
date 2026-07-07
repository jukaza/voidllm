import { LOCAL_STORAGE_KEY } from '../lib/constants'
import { handleUnauthorized } from '../lib/authSession'

async function parseJSON<T>(res: Response): Promise<T> {
  const text = await res.text()
  try {
    return JSON.parse(text) as T
  } catch {
    if (text.trimStart().toLowerCase().startsWith('<!doctype') || text.trimStart().startsWith('<')) {
      throw new Error(
        'Server returned HTML instead of JSON. Rebuild and restart tavo-server (go build -o tavo-server ./cmd/tavo).',
      )
    }
    throw new Error('Invalid JSON response from server')
  }
}

const apiClient = async <T>(endpoint: string, options?: RequestInit): Promise<T> => {
  const key = localStorage.getItem(LOCAL_STORAGE_KEY) ?? ''
  const res = await fetch(`/api/v1${endpoint}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${key}`,
      ...options?.headers,
    },
  })

  if (res.status === 401) {
    handleUnauthorized()
    throw new Error('Session expired')
  }

  if (!res.ok) {
    try {
      const error = await parseJSON<{ error?: { message?: string } }>(res)
      throw new Error(error?.error?.message ?? res.statusText)
    } catch (e) {
      if (e instanceof Error && e.message !== res.statusText) throw e
      throw new Error(res.statusText)
    }
  }

  if (res.status === 204) {
    // DELETE endpoints return no body. Callers must use apiClient<void>().
    return undefined as T
  }

  return parseJSON<T>(res)
}

export default apiClient
