export interface OAuthProviderDraft {
  enabled: boolean
  client_id: string
  client_secret: string
}

export interface SettingsDraft {
  site_subtitle: string
  support_email: string
  support_telegram: string
  support_discord: string
  doc_url: string
  turnstile_enabled: boolean
  turnstile_site_key: string
  turnstile_secret_key: string
  oauth_google: OAuthProviderDraft
  oauth_github: OAuthProviderDraft
  oauth_oidc: OAuthProviderDraft & { issuer_url: string }
  enforce_balance: boolean
  initial_wallet_balance: number
  public_catalog_enabled: boolean
  playground_enabled: boolean
  require_terms_on_login: boolean
}

export const DEFAULT_SETTINGS_DRAFT: SettingsDraft = {
  site_subtitle: '',
  support_email: '',
  support_telegram: '',
  support_discord: '',
  doc_url: '',
  turnstile_enabled: false,
  turnstile_site_key: '',
  turnstile_secret_key: '',
  oauth_google: { enabled: false, client_id: '', client_secret: '' },
  oauth_github: { enabled: false, client_id: '', client_secret: '' },
  oauth_oidc: { enabled: false, client_id: '', client_secret: '', issuer_url: '' },
  enforce_balance: false,
  initial_wallet_balance: 0,
  public_catalog_enabled: true,
  playground_enabled: true,
  require_terms_on_login: false,
}

export const SETTINGS_DRAFT_STORAGE_KEY = 'voidllm.admin_settings.draft.v1'