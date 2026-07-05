import { useEffect, useState } from 'react'
import { Input } from '../../components/ui/Input'
import { Textarea } from '../../components/ui/Textarea'
import { useAdminSiteSettings, useUpdateSiteConfig } from '../../hooks/useSiteConfig'
import { useToast } from '../../hooks/useToast'
import { useTranslation } from '../../lib/i18n'
import { LiveBadge } from './components/PreviewBadge'
import { SettingsSectionCard } from './components/SettingsSectionCard'
import { contactHref } from '../../lib/contactLinks'
import { SiteLogoField } from './components/SiteLogoField'
import { SettingsTabFooter } from './components/SettingsTabFooter'

export function GeneralSettingsTab() {
  const { t } = useTranslation()
  const { toast } = useToast()
  const { data } = useAdminSiteSettings()
  const updateSite = useUpdateSiteConfig()

  const [systemName, setSystemName] = useState('')
  const [siteSubtitle, setSiteSubtitle] = useState('')
  const [serverAddress, setServerAddress] = useState('')
  const [footer, setFooter] = useState('')
  const [homePageContent, setHomePageContent] = useState('')
  const [supportZalo, setSupportZalo] = useState('')
  const [supportTelegram, setSupportTelegram] = useState('')
  const [docUrl, setDocUrl] = useState('')

  useEffect(() => {
    if (!data) return
    setSystemName(data.system_name)
    setSiteSubtitle(data.site_subtitle ?? '')
    setServerAddress(data.server_address)
    setFooter(data.footer)
    setHomePageContent(data.home_page_content)
    setSupportZalo(data.support_zalo ?? '')
    setSupportTelegram(data.support_telegram ?? '')
    setDocUrl(data.doc_url ?? '')
  }, [data])

  function save() {
    updateSite.mutate(
      {
        system_name: systemName.trim(),
        site_subtitle: siteSubtitle.trim(),
        server_address: serverAddress.trim(),
        footer: footer.trim(),
        home_page_content: homePageContent.trim(),
        support_zalo: supportZalo.trim(),
        support_telegram: supportTelegram.trim(),
        doc_url: docUrl.trim(),
      },
      {
        onSuccess: () => toast({ variant: 'success', message: t('common.saved') }),
        onError: (err) => toast({ variant: 'error', message: err.message }),
      },
    )
  }

  const zaloHref = contactHref(supportZalo)
  const telegramHref = contactHref(supportTelegram)
  const docHref = contactHref(docUrl)

  return (
    <div className="space-y-6">
      <SettingsSectionCard
        title={t('settings.branding_title')}
        description={t('settings.branding_desc')}
        badge={<LiveBadge />}
      >
        <Input
          label={t('settings.system_name')}
          value={systemName}
          onChange={(e) => setSystemName(e.target.value)}
          description={t('settings.system_name_hint')}
          required
        />
        <Input
          label={t('settings.site_subtitle')}
          value={siteSubtitle}
          onChange={(e) => setSiteSubtitle(e.target.value)}
          description={t('settings.site_subtitle_hint')}
        />
        <SiteLogoField logo={data?.logo ?? ''} />
        <Input
          label={t('settings.footer')}
          value={footer}
          onChange={(e) => setFooter(e.target.value)}
          description={t('settings.footer_hint')}
        />
      </SettingsSectionCard>

      <SettingsSectionCard
        title={t('settings.homepage_title')}
        description={t('settings.homepage_desc')}
        badge={<LiveBadge />}
      >
        <Textarea
          label={t('settings.home_page_content')}
          value={homePageContent}
          onChange={(e) => setHomePageContent(e.target.value)}
          rows={12}
          description={t('settings.home_page_content_hint')}
        />
      </SettingsSectionCard>

      <SettingsSectionCard
        title={t('settings.contact_title')}
        description={t('settings.contact_desc')}
        badge={<LiveBadge />}
      >
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div className="space-y-1">
            <Input
              label={t('settings.support_zalo')}
              value={supportZalo}
              onChange={(e) => setSupportZalo(e.target.value)}
              placeholder="https://zalo.me/g/your-group"
              description={t('settings.support_zalo_hint')}
            />
            {zaloHref && (
              <a
                href={zaloHref}
                target="_blank"
                rel="noopener noreferrer"
                className="text-xs text-accent no-underline hover:opacity-80"
              >
                {t('settings.contact_link_preview')}
              </a>
            )}
          </div>
          <div className="space-y-1">
            <Input
              label={t('settings.support_telegram')}
              value={supportTelegram}
              onChange={(e) => setSupportTelegram(e.target.value)}
              placeholder="https://t.me/your-group"
              description={t('settings.support_telegram_hint')}
            />
            {telegramHref && (
              <a
                href={telegramHref}
                target="_blank"
                rel="noopener noreferrer"
                className="text-xs text-accent no-underline hover:opacity-80"
              >
                {t('settings.contact_link_preview')}
              </a>
            )}
          </div>
        </div>
        <Input
          label={t('settings.doc_url')}
          value={docUrl}
          onChange={(e) => setDocUrl(e.target.value)}
          placeholder="https://docs.example.com"
          description={t('settings.doc_url_hint')}
        />
        {docHref && (
          <a
            href={docHref}
            target="_blank"
            rel="noopener noreferrer"
            className="text-xs text-accent no-underline hover:opacity-80"
          >
            {t('settings.contact_link_preview')}
          </a>
        )}
      </SettingsSectionCard>

      <SettingsSectionCard
        title={t('settings.integration_title')}
        description={t('settings.integration_desc')}
        badge={<LiveBadge />}
      >
        <Input
          label={t('settings.server_address')}
          value={serverAddress}
          onChange={(e) => setServerAddress(e.target.value)}
          placeholder="https://api.example.com"
          description={t('settings.server_address_hint')}
        />
      </SettingsSectionCard>

      <SettingsTabFooter mode="live" loading={updateSite.isPending} onSave={save} />
    </div>
  )
}