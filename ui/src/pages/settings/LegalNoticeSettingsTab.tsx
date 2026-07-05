import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Button } from '../../components/ui/Button'
import { Textarea } from '../../components/ui/Textarea'
import { Toggle } from '../../components/ui/Toggle'
import { useAdminSiteSettings, useUpdateSiteConfig } from '../../hooks/useSiteConfig'
import { useToast } from '../../hooks/useToast'
import { useTranslation } from '../../lib/i18n'
import type { SiteAnnouncement } from '../../lib/announcements'
import { LiveBadge } from './components/PreviewBadge'
import { SettingsSectionCard } from './components/SettingsSectionCard'
import { SettingsTabFooter } from './components/SettingsTabFooter'
import { NoticeListEditor } from './NoticeListEditor'
import { useSettingsDraft } from './useSettingsDraft'

export function LegalNoticeSettingsTab() {
  const { t } = useTranslation()
  const { toast } = useToast()
  const { data } = useAdminSiteSettings()
  const updateSite = useUpdateSiteConfig()
  const { draft, setDraft } = useSettingsDraft()

  const [userAgreement, setUserAgreement] = useState('')
  const [privacyPolicy, setPrivacyPolicy] = useState('')
  const [announcements, setAnnouncements] = useState<SiteAnnouncement[]>([])
  const [noticeEnabled, setNoticeEnabled] = useState(false)

  useEffect(() => {
    if (!data) return
    setUserAgreement(data.user_agreement)
    setPrivacyPolicy(data.privacy_policy)
    setAnnouncements(data.announcements ?? [])
    setNoticeEnabled(data.notice_enabled)
  }, [data])

  function save() {
    const valid = announcements
      .map((item) => ({
        ...item,
        content: item.content.trim(),
        extra: item.extra?.trim() ?? '',
      }))
      .filter((item) => item.content.length > 0)

    updateSite.mutate(
      {
        user_agreement: userAgreement.trim(),
        privacy_policy: privacyPolicy.trim(),
        announcements: valid,
        notice_enabled: noticeEnabled,
      },
      {
        onSuccess: () => toast({ variant: 'success', message: t('common.saved') }),
        onError: (err) => toast({ variant: 'error', message: err.message }),
      },
    )
  }

  return (
    <div className="space-y-6">
      <SettingsSectionCard
        title={t('settings.legal_docs')}
        description={t('settings.legal_docs_hint')}
        badge={<LiveBadge />}
      >
        <Textarea
          label={t('settings.user_agreement')}
          value={userAgreement}
          onChange={(e) => setUserAgreement(e.target.value)}
          rows={14}
          description={t('settings.user_agreement_hint')}
        />
        <Textarea
          label={t('settings.privacy_policy')}
          value={privacyPolicy}
          onChange={(e) => setPrivacyPolicy(e.target.value)}
          rows={14}
          description={t('settings.privacy_policy_hint')}
        />
        <div className="flex flex-wrap gap-2">
          <Link to="/legal/user-agreement" target="_blank" rel="noreferrer">
            <Button size="sm" variant="secondary">
              {t('settings.preview_terms')}
            </Button>
          </Link>
          <Link to="/legal/privacy-policy" target="_blank" rel="noreferrer">
            <Button size="sm" variant="secondary">
              {t('settings.preview_privacy')}
            </Button>
          </Link>
        </div>
        <Toggle
          checked={draft.require_terms_on_login}
          onChange={(v) => setDraft({ require_terms_on_login: v })}
          label={t('settings.require_terms_on_login')}
        />
        <p className="text-xs text-text-tertiary -mt-2">{t('settings.require_terms_on_login_hint')}</p>
      </SettingsSectionCard>

      <SettingsSectionCard
        title={t('settings.notice')}
        description={t('settings.notice_hint')}
        badge={<LiveBadge />}
      >
        <Toggle checked={noticeEnabled} onChange={setNoticeEnabled} label={t('settings.notice_enabled')} />
        <NoticeListEditor items={announcements} onChange={setAnnouncements} />
      </SettingsSectionCard>

      <SettingsTabFooter mode="live" loading={updateSite.isPending} onSave={save} />
    </div>
  )
}