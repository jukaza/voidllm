export const LOCAL_STORAGE_KEY = 'tavo_session'
export const LAST_READ_NOTICE_KEY = 'tavo_last_read_notice'

/** Maps backend key_type values to their display prefixes. */
export const KEY_PREFIXES: Record<string, string> = {
  user_key: 'sk-',
  session_key: 'sk-',
} as const

export type KeyType = 'user_key' | 'session_key'
