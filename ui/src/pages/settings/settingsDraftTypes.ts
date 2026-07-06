export interface OAuthProviderDraft {
  enabled: boolean
  client_id: string
  client_secret: string
}

export interface SettingsDraft {
  require_terms_on_login: boolean
}

export const DEFAULT_SETTINGS_DRAFT: SettingsDraft = {
  require_terms_on_login: false,
}

export const SETTINGS_DRAFT_STORAGE_KEY = 'voidllm.admin_settings.draft.v1'