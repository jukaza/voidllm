import type { PublicAuthConfig } from '../hooks/useSecuritySettings'
import type { TranslationKey } from './i18n'

export const OAUTH_PROVIDERS = [
  { id: 'google' as const, labelKey: 'account.oauth_google' as const },
  { id: 'github' as const, labelKey: 'account.oauth_github' as const },
] as const

export type OAuthProviderId = (typeof OAUTH_PROVIDERS)[number]['id']

export function oauthProviderLabelKey(id: OAuthProviderId): TranslationKey {
  return OAUTH_PROVIDERS.find((p) => p.id === id)!.labelKey
}

/** True when at least one OAuth provider is configured for account linking. */
export function hasOAuthLinking(config?: PublicAuthConfig): boolean {
  if (!config?.oauth) return false
  return Boolean(config.oauth.google?.enabled || config.oauth.github?.enabled)
}

export function hasOAuthLogin(config?: PublicAuthConfig): boolean {
  if (!config?.oauth) return false
  return Boolean(config.oauth.google?.login || config.oauth.github?.login)
}

export function hasOAuthSignup(config?: PublicAuthConfig): boolean {
  if (!config?.oauth) return false
  return Boolean(config.oauth.google?.signup || config.oauth.github?.signup)
}