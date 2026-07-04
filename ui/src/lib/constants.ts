export const LOCAL_STORAGE_KEY = 'voidllm_session'

/** Maps backend key_type values to their display prefixes. */
export const KEY_PREFIXES: Record<string, string> = {
  user_key: 'vl_uk_',
  session_key: 'vl_sk_',
} as const

export type KeyType = 'user_key' | 'session_key'
