import { useEffect, useState } from 'react'
import { Input } from '../../components/ui/Input'
import { Textarea } from '../../components/ui/Textarea'
import { useAdminSiteSettings, useUpdateSiteConfig } from '../../hooks/useSiteConfig'
import { useToast } from '../../hooks/useToast'
import { useTranslation } from '../../lib/i18n'
import { LiveBadge } from './components/PreviewBadge'
import { SettingsSectionCard } from './components/SettingsSectionCard'
import { SettingsTabFooter } from './components/SettingsTabFooter'
import { useSettingsDraft } from './useSettingsDraft'

export function GeneralSettingsTab() {
  const { t } = useTranslation()
  const { toast } = useToast()
  const { data } = useAdminSiteSettings()
  const updateSite = useUpdateSiteConfig()
  const { draft, setDraft } = useSettingsDraft()

  const [systemName, setSystemName] = useState('')
  const [logo, setLogo] = useState('')
  const [serverAddress, setServerAddress] = useState('')
  const [footer, setFooter] = useState('')
  const [about, setAbout] = useState('')
  const [homePageContent, setHomePageContent] = useState('')

  useEffect(() => {
    if (!data) return
    setSystemName(data.system_name)
    setLogo(data.logo)
    setServerAddress(data.server_address)
    setFooter(data.footer)
    setAbout(data.about)
    setHomePageContent(data.home_page_content)
  }, [data])

  function save() {
    updateSite.mutate(
      {
        system_name: systemName.trim(),
        logo: logo.trim(),
        server_address: serverAddress.trim(),
        footer: footer.trim(),
        about: about.trim(),
        home_page_content: homePageContent.trim(),
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
          value={draft.site_subtitle}
          onChange={(e) => setDraft({ site_subtitle: e.target.value })}
          description={t('settings.site_subtitle_hint')}
        />
        <Input
          label={t('settings.logo')}
          value={logo}
          onChange={(e) => setLogo(e.target.value)}
          placeholder="/logo.svg"
          description={t('settings.logo_hint')}
        />
        {logo.trim() && (
          <div className="flex items-center gap-3 rounded-md border border-border bg-bg-tertiary p-3">
            <img
              src={logo.trim()}
              alt=""
              className="h-10 w-10 object-contain"
              onError={(e) => {
                e.currentTarget.style.display = 'none'
              }}
            />
            <span className="text-xs text-text-tertiary">{t('settings.logo_preview')}</span>
          </div>
        )}
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
      </SettingsSectionCard>

      <SettingsSectionCard title={t('settings.homepage_title')} description={t('settings.homepage_desc')}>
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
        <Input
          label={t('settings.support_email')}
          value={draft.support_email}
          onChange={(e) => setDraft({ support_email: e.target.value })}
          placeholder="support@example.com"
        />
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Input
            label={t('settings.support_telegram')}
            value={draft.support_telegram}
            onChange={(e) => setDraft({ support_telegram: e.target.value })}
            placeholder="@voidllm"
          />
          <Input
            label={t('settings.support_discord')}
            value={draft.support_discord}
            onChange={(e) => setDraft({ support_discord: e.target.value })}
            placeholder="https://discord.gg/..."
          />
        </div>
        <Input
          label={t('settings.doc_url')}
          value={draft.doc_url}
          onChange={(e) => setDraft({ doc_url: e.target.value })}
          placeholder="https://docs.example.com"
          description={t('settings.doc_url_hint')}
        />
      </SettingsSectionCard>

      <SettingsTabFooter mode="live" loading={updateSite.isPending} onSave={save} />
    </div>
  )
}