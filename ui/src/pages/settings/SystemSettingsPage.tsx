import { useEffect, useState } from 'react'
import { PageHeader } from '../../components/ui/PageHeader'
import { Input } from '../../components/ui/Input'
import { Textarea } from '../../components/ui/Textarea'
import { Button } from '../../components/ui/Button'
import { Toggle } from '../../components/ui/Toggle'
import TabSwitcher from '../../components/ui/TabSwitcher'
import { useAdminSiteSettings, useUpdateSiteConfig } from '../../hooks/useSiteConfig'
import { Banner } from '../../components/ui/Banner'
import { useToast } from '../../hooks/useToast'
import { useTranslation } from '../../lib/i18n'
import { NoticeListEditor } from './NoticeListEditor'
import { PaymentSettingsTab } from './PaymentSettingsTab'
import { EmailSettingsTab } from './EmailSettingsTab'
import type { SiteAnnouncement } from '../../lib/announcements'

type SettingsTab = 'system' | 'legal' | 'notice' | 'payment' | 'email'

function SectionCard({ title, description, children }: { title: string; description?: string; children: React.ReactNode }) {
  return (
    <div className="rounded-lg border border-border bg-bg-secondary">
      <div className="px-6 py-4 border-b border-border">
        <h2 className="text-sm font-semibold text-text-primary">{title}</h2>
        {description && <p className="mt-1 text-xs text-text-tertiary">{description}</p>}
      </div>
      <div className="p-6 space-y-5">{children}</div>
    </div>
  )
}

export default function SystemSettingsPage() {
  const { t } = useTranslation()
  const { data, isLoading, isError, error, refetch } = useAdminSiteSettings()
  const updateSite = useUpdateSiteConfig()
  const { toast } = useToast()
  const [tab, setTab] = useState<SettingsTab>('system')

  const [systemName, setSystemName] = useState('')
  const [logo, setLogo] = useState('')
  const [serverAddress, setServerAddress] = useState('')
  const [footer, setFooter] = useState('')
  const [about, setAbout] = useState('')
  const [homePageContent, setHomePageContent] = useState('')
  const [userAgreement, setUserAgreement] = useState('')
  const [privacyPolicy, setPrivacyPolicy] = useState('')
  const [announcements, setAnnouncements] = useState<SiteAnnouncement[]>([])
  const [noticeEnabled, setNoticeEnabled] = useState(false)
  const [registerEnabled, setRegisterEnabled] = useState(true)

  useEffect(() => {
    if (!data) return
    setSystemName(data.system_name)
    setLogo(data.logo)
    setServerAddress(data.server_address)
    setFooter(data.footer)
    setAbout(data.about)
    setHomePageContent(data.home_page_content)
    setUserAgreement(data.user_agreement)
    setPrivacyPolicy(data.privacy_policy)
    setAnnouncements(data.announcements ?? [])
    setNoticeEnabled(data.notice_enabled)
    setRegisterEnabled(data.register_enabled)
  }, [data])

  function save(payload: Parameters<typeof updateSite.mutate>[0]) {
    updateSite.mutate(payload, {
      onSuccess: () => toast({ variant: 'success', message: t('common.saved') }),
      onError: (err) => toast({ variant: 'error', message: err.message }),
    })
  }

  const tabs = [
    { key: 'system', label: t('settings.tab_system') },
    { key: 'payment', label: t('settings.tab_payment') },
    { key: 'email', label: t('settings.tab_email') },
    { key: 'legal', label: t('settings.tab_legal') },
    { key: 'notice', label: t('settings.tab_notice') },
  ]

  if (isLoading) {
    return (
      <>
        <PageHeader title={t('settings.title')} description={t('settings.description')} />
        <p className="text-sm text-text-tertiary">{t('common.loading')}</p>
      </>
    )
  }

  if (isError || !data) {
    return (
      <>
        <PageHeader title={t('settings.title')} description={t('settings.description')} />
        <Banner
          variant="error"
          title={t('settings.load_error')}
          description={error?.message ?? t('settings.load_error_hint')}
        />
        <Button className="mt-4" variant="secondary" onClick={() => void refetch()}>
          {t('common.refresh')}
        </Button>
      </>
    )
  }

  return (
    <>
      <PageHeader title={t('settings.title')} description={t('settings.description')} />

      <TabSwitcher tabs={tabs} activeKey={tab} onChange={(key) => setTab(key as SettingsTab)} />

      {tab === 'system' && (
        <SectionCard title={t('settings.system_info')} description={t('settings.system_info_hint')}>
          <Input
            label={t('settings.system_name')}
            value={systemName}
            onChange={(e) => setSystemName(e.target.value)}
            description={t('settings.system_name_hint')}
            required
          />
          <Input
            label={t('settings.logo')}
            value={logo}
            onChange={(e) => setLogo(e.target.value)}
            placeholder="/logo.svg"
            description={t('settings.logo_hint')}
          />
          <Input
            label={t('settings.server_address')}
            value={serverAddress}
            onChange={(e) => setServerAddress(e.target.value)}
            placeholder="https://api.example.com"
            description={t('settings.server_address_hint')}
          />
          <Input
            label={t('settings.footer')}
            value={footer}
            onChange={(e) => setFooter(e.target.value)}
            description={t('settings.footer_hint')}
          />
          <Textarea
            label={t('settings.about')}
            value={about}
            onChange={(e) => setAbout(e.target.value)}
            rows={4}
            description={t('settings.about_hint')}
          />
          <Textarea
            label={t('settings.home_page_content')}
            value={homePageContent}
            onChange={(e) => setHomePageContent(e.target.value)}
            rows={12}
            description={t('settings.home_page_content_hint')}
          />
          <Toggle
            checked={registerEnabled}
            onChange={setRegisterEnabled}
            label={t('settings.register_enabled')}
          />
          <div className="flex justify-end">
            <Button
              loading={updateSite.isPending}
              onClick={() =>
                save({
                  system_name: systemName.trim(),
                  logo: logo.trim(),
                  server_address: serverAddress.trim(),
                  footer: footer.trim(),
                  about: about.trim(),
                  home_page_content: homePageContent.trim(),
                  register_enabled: registerEnabled,
                })
              }
            >
              {t('common.save')}
            </Button>
          </div>
        </SectionCard>
      )}

      {tab === 'payment' && <PaymentSettingsTab />}

      {tab === 'email' && <EmailSettingsTab />}

      {tab === 'legal' && (
        <SectionCard title={t('settings.legal_docs')} description={t('settings.legal_docs_hint')}>
          <Textarea
            label={t('settings.user_agreement')}
            value={userAgreement}
            onChange={(e) => setUserAgreement(e.target.value)}
            rows={16}
            description={t('settings.user_agreement_hint')}
          />
          <Textarea
            label={t('settings.privacy_policy')}
            value={privacyPolicy}
            onChange={(e) => setPrivacyPolicy(e.target.value)}
            rows={16}
            description={t('settings.privacy_policy_hint')}
          />
          <div className="flex justify-end">
            <Button
              loading={updateSite.isPending}
              onClick={() =>
                save({
                  user_agreement: userAgreement.trim(),
                  privacy_policy: privacyPolicy.trim(),
                })
              }
            >
              {t('common.save')}
            </Button>
          </div>
        </SectionCard>
      )}

      {tab === 'notice' && (
        <SectionCard title={t('settings.notice')} description={t('settings.notice_hint')}>
          <Toggle
            checked={noticeEnabled}
            onChange={setNoticeEnabled}
            label={t('settings.notice_enabled')}
          />
          <NoticeListEditor items={announcements} onChange={setAnnouncements} />
          <div className="flex justify-end">
            <Button
              loading={updateSite.isPending}
              onClick={() => {
                const valid = announcements
                  .map((item) => ({
                    ...item,
                    content: item.content.trim(),
                    extra: item.extra?.trim() ?? '',
                  }))
                  .filter((item) => item.content.length > 0)
                save({
                  announcements: valid,
                  notice_enabled: noticeEnabled,
                })
              }}
            >
              {t('common.save')}
            </Button>
          </div>
        </SectionCard>
      )}
    </>
  )
}