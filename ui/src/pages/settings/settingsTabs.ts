import type { TranslationKey } from '../../lib/i18n'

export type SettingsTabKey =
  | 'general'
  | 'security'
  | 'features'
  | 'payment'
  | 'legal'
  | 'backup'

export const SETTINGS_TAB_KEYS: SettingsTabKey[] = [
  'general',
  'security',
  'features',
  'payment',
  'legal',
  'backup',
]

export function isSettingsTabKey(value: string | null): value is SettingsTabKey {
  return SETTINGS_TAB_KEYS.includes(value as SettingsTabKey)
}

export const SETTINGS_TAB_I18N: Record<SettingsTabKey, TranslationKey> = {
  general: 'settings.tab_general',
  security: 'settings.tab_security',
  features: 'settings.tab_features',
  payment: 'settings.tab_payment',
  legal: 'settings.tab_legal_notice',
  backup: 'settings.tab_backup',
}

/** Tabs whose primary save path is local draft (no dedicated admin API yet). */
export const PREVIEW_TAB_KEYS = new Set<SettingsTabKey>(['security', 'features', 'backup'])