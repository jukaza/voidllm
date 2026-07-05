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
  oauth_oidc: OAuthBindingDraft
  record_ip: boolean
  sessions: SessionDraft[]
}

export const DEFAULT_ACCOUNT_DRAFT: AccountDraft = {
  display_name: '',
  two_fa_enabled: false,
  oauth_google: { bound: false },
  oauth_github: { bound: true, value: 'octocat' },
  oauth_oidc: { bound: false },
  record_ip: true,
  sessions: [
    { id: 's1', ip: '203.0.113.45', lastActive: '2 minutes ago', device: 'Chrome on macOS', current: true },
    { id: 's2', ip: '198.51.100.12', lastActive: 'yesterday', device: 'Safari on iPhone' },
  ],
}

export const ACCOUNT_DRAFT_STORAGE_KEY = 'voidllm.account.draft.v1'