import { useTranslation } from '../../../lib/i18n'
import { SetupGuideBlock } from './SecuritySetupGuide'

const R2_LINKS = {
  overview: 'https://dash.cloudflare.com/?to=/:account/r2/overview',
  apiTokens: 'https://dash.cloudflare.com/?to=/:account/r2/api-tokens',
  docs: 'https://developers.cloudflare.com/r2/api/s3/api/',
} as const

export function BackupR2SetupGuide() {
  const { t } = useTranslation()
  return (
    <div className="mb-4 space-y-3">
      <SetupGuideBlock
        linkHref={R2_LINKS.overview}
        linkLabel={t('settings.backup_r2_link_bucket')}
        hint={t('settings.backup_r2_step_bucket')}
      />
      <SetupGuideBlock
        linkHref={R2_LINKS.apiTokens}
        linkLabel={t('settings.backup_r2_link_token')}
        hint={t('settings.backup_r2_step_token')}
      />
      <div className="rounded-md border border-border/70 bg-bg-tertiary/30 px-3 py-2.5 space-y-2">
        <p className="text-xs font-medium text-text-secondary">{t('settings.backup_r2_step_fill_title')}</p>
        <ul className="list-disc space-y-1 pl-4 text-xs text-text-tertiary leading-relaxed">
          <li>{t('settings.backup_r2_fill_endpoint')}</li>
          <li>{t('settings.backup_r2_fill_region')}</li>
          <li>{t('settings.backup_r2_fill_bucket')}</li>
          <li>{t('settings.backup_r2_fill_prefix')}</li>
          <li>{t('settings.backup_r2_fill_keys')}</li>
          <li>{t('settings.backup_r2_fill_path_style')}</li>
        </ul>
        <a
          href={R2_LINKS.docs}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-1 text-xs font-medium text-accent no-underline hover:opacity-80"
        >
          {t('settings.backup_r2_link_docs')}
          <span aria-hidden>↗</span>
        </a>
      </div>
    </div>
  )
}