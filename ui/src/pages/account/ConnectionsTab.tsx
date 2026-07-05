import { Button } from '../../components/ui/Button'
import { Badge } from '../../components/ui/Badge'
import { useToast } from '../../hooks/useToast'
import { useTranslation } from '../../lib/i18n'
import { PreviewBadge } from '../settings/components/PreviewBadge'
import { SettingsSectionCard } from '../settings/components/SettingsSectionCard'
import { SettingsTabFooter } from '../settings/components/SettingsTabFooter'
import type { OAuthBindingDraft } from './accountDraftTypes'
import { useAccountDraft } from './useAccountDraft'

type OAuthKey = 'oauth_google' | 'oauth_github' | 'oauth_oidc'

const PROVIDERS: { key: OAuthKey; labelKey: 'account.oauth_google' | 'account.oauth_github' | 'account.oauth_oidc'; abbr: string }[] = [
  { key: 'oauth_google', labelKey: 'account.oauth_google', abbr: 'Go' },
  { key: 'oauth_github', labelKey: 'account.oauth_github', abbr: 'GH' },
  { key: 'oauth_oidc', labelKey: 'account.oauth_oidc', abbr: 'SS' },
]

export function ConnectionsTab() {
  const { t } = useTranslation()
  const { toast } = useToast()
  const { draft, setDraft } = useAccountDraft()

  function toggleBinding(key: OAuthKey, label: string, current: OAuthBindingDraft) {
    const nextBound = !current.bound
    setDraft({
      [key]: {
        bound: nextBound,
        value: nextBound ? current.value || 'connected-user' : undefined,
      },
    })
    toast({
      variant: 'success',
      message: nextBound
        ? t('account.oauth_linked_demo', { provider: label })
        : t('account.oauth_unlinked_demo', { provider: label }),
    })
  }

  function save() {
    toast({ variant: 'success', message: t('account.saved_preview') })
  }

  return (
    <div className="space-y-6">
      <SettingsSectionCard
        title={t('account.connections_title')}
        description={t('account.connections_desc')}
        badge={<PreviewBadge />}
      >
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          {PROVIDERS.map((provider) => {
            const binding = draft[provider.key]
            return (
              <div
                key={provider.key}
                className="flex items-center justify-between gap-3 rounded-md border border-border p-4"
              >
                <div className="flex min-w-0 items-center gap-3">
                  <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-bg-tertiary text-[10px] font-semibold">
                    {provider.abbr}
                  </div>
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium">{t(provider.labelKey)}</span>
                      {binding.bound && (
                        <Badge variant="success" className="text-[10px]">
                          {t('account.status_on')}
                        </Badge>
                      )}
                    </div>
                    {binding.bound && binding.value && (
                      <p className="truncate text-xs text-text-tertiary">
                        {t('account.oauth_linked', { value: binding.value })}
                      </p>
                    )}
                  </div>
                </div>
                <Button
                  size="sm"
                  variant={binding.bound ? 'ghost' : 'secondary'}
                  className="shrink-0"
                  onClick={() => toggleBinding(provider.key, t(provider.labelKey), binding)}
                >
                  {binding.bound ? t('account.oauth_unlink') : t('account.oauth_link')}
                </Button>
              </div>
            )
          })}
        </div>
        <p className="text-xs text-text-tertiary">{t('account.connections_hint')}</p>
      </SettingsSectionCard>

      <SettingsTabFooter mode="preview" onSave={save} />
    </div>
  )
}