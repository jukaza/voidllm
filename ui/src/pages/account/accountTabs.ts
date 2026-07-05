import type { TranslationKey } from '../../lib/i18n'

export type AccountTabKey = 'profile' | 'security' | 'connections' | 'preferences'

export const ACCOUNT_TAB_KEYS: AccountTabKey[] = [
  'profile',
  'security',
  'connections',
  'preferences',
]

export function isAccountTabKey(value: string | null): value is AccountTabKey {
  return ACCOUNT_TAB_KEYS.includes(value as AccountTabKey)
}

export const ACCOUNT_TAB_I18N: Record<AccountTabKey, TranslationKey> = {
  profile: 'account.tab_profile',
  security: 'account.tab_security',
  connections: 'account.tab_connections',
  preferences: 'account.tab_preferences',
}