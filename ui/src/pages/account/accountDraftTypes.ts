export interface OAuthBindingDraft {
  bound: boolean
  value?: string
}

export interface SessionDraft {
  id: string
  ip: string
  lastActive: string
  device: string
  current?: boolean
}

export interface AccountDraft {
  display_name: string
  two_fa_enabled: boolean
  oauth_google: OAuthBindingDraft
  oauth_github: OAuthBindingDraft
  record_ip: boolean
  sessions: SessionDraft[]
}

export const DEFAULT_ACCOUNT_DRAFT: AccountDraft = {
  display_name: '',
  two_fa_enabled: false,
  oauth_google: { bound: false },
  oauth_github: { bound: false },
  record_ip: true,
  sessions: [],
}

export const ACCOUNT_DRAFT_STORAGE_KEY = 'voidllm.account.draft.v1'