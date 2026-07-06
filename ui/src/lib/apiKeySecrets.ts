const STORAGE_PREFIX = 'voidllm:api-key:'

export function rememberApiKey(id: string, key: string): void {
  if (!id || !key) return
  try {
    sessionStorage.setItem(STORAGE_PREFIX + id, key)
  } catch {
    /* ignore quota / private mode */
  }
}

export function getRememberedApiKey(id: string): string | null {
  if (!id) return null
  try {
    return sessionStorage.getItem(STORAGE_PREFIX + id)
  } catch {
    return null
  }
}

export function forgetApiKey(id: string): void {
  if (!id) return
  try {
    sessionStorage.removeItem(STORAGE_PREFIX + id)
  } catch {
    /* ignore */
  }
}