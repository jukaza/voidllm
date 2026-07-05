import { useRef, useState } from 'react'
import { Button } from '../../../components/ui/Button'
import { useResetSiteLogo, useUploadSiteLogo } from '../../../hooks/useSiteConfig'
import { useToast } from '../../../hooks/useToast'
import { useTranslation } from '../../../lib/i18n'
import { DEFAULT_SITE_LOGO, isUploadedSiteLogo, logoSrc } from '../../../lib/siteLogo'

interface SiteLogoFieldProps {
  logo: string
}

export function SiteLogoField({ logo }: SiteLogoFieldProps) {
  const { t } = useTranslation()
  const { toast } = useToast()
  const fileRef = useRef<HTMLInputElement>(null)
  const uploadLogo = useUploadSiteLogo()
  const resetLogo = useResetSiteLogo()
  const [cacheBust, setCacheBust] = useState(0)

  const displayLogo = logo.trim() || DEFAULT_SITE_LOGO
  const showReset = isUploadedSiteLogo(displayLogo)

  function pickFile() {
    fileRef.current?.click()
  }

  async function handleFileChange(file: File | undefined) {
    if (!file) return
    try {
      await uploadLogo.mutateAsync(file)
      setCacheBust(Date.now())
      toast({ variant: 'success', message: t('settings.logo_uploaded') })
    } catch (err) {
      toast({
        variant: 'error',
        message: err instanceof Error ? err.message : t('settings.logo_upload_error'),
      })
    } finally {
      if (fileRef.current) fileRef.current.value = ''
    }
  }

  async function handleReset() {
    try {
      await resetLogo.mutateAsync()
      setCacheBust(Date.now())
      toast({ variant: 'success', message: t('settings.logo_reset') })
    } catch (err) {
      toast({
        variant: 'error',
        message: err instanceof Error ? err.message : t('settings.logo_reset_error'),
      })
    }
  }

  const busy = uploadLogo.isPending || resetLogo.isPending

  return (
    <div className="space-y-3">
      <div>
        <p className="text-sm font-medium text-text-primary">{t('settings.logo')}</p>
        <p className="mt-0.5 text-xs text-text-tertiary">{t('settings.logo_upload_hint')}</p>
      </div>

      <div className="flex flex-wrap items-center gap-4 rounded-md border border-border bg-bg-tertiary p-4">
        <img
          src={logoSrc(displayLogo, cacheBust)}
          alt=""
          className="h-14 w-14 object-contain rounded-md bg-bg-primary p-1"
          onError={(e) => {
            e.currentTarget.src = logoSrc(DEFAULT_SITE_LOGO, cacheBust)
          }}
        />
        <div className="flex flex-wrap gap-2">
          <Button type="button" variant="secondary" size="sm" loading={uploadLogo.isPending} onClick={pickFile}>
            {t('settings.logo_upload')}
          </Button>
          {showReset && (
            <Button type="button" variant="ghost" size="sm" loading={resetLogo.isPending} disabled={busy} onClick={() => void handleReset()}>
              {t('settings.logo_reset_default')}
            </Button>
          )}
        </div>
      </div>

      <input
        ref={fileRef}
        type="file"
        accept="image/png,image/jpeg,image/webp,image/svg+xml,.png,.jpg,.jpeg,.webp,.svg"
        className="hidden"
        onChange={(e) => void handleFileChange(e.target.files?.[0])}
      />
    </div>
  )
}