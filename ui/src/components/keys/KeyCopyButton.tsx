import { useState } from 'react'
import apiClient from '../../api/client'
import { KeyHint } from '../ui/KeyHint'
import { useTranslation } from '../../lib/i18n'
import { useToast } from '../../hooks/useToast'

export function KeyCopyButton({ keyId, hint }: { keyId: string; hint: string }) {
  const { t } = useTranslation()
  const { toast } = useToast()
  const [copying, setCopying] = useState(false)

  async function handleCopy() {
    if (copying) return
    setCopying(true)
    try {
      const res = await apiClient<{ key: string }>(`/keys/${keyId}/reveal`)
      await navigator.clipboard.writeText(res.key)
      toast({ variant: 'success', message: t('common.copied') })
    } catch (e) {
      const msg = e instanceof Error ? e.message : t('keys.copy_failed')
      toast({ variant: 'error', message: msg })
    } finally {
      setCopying(false)
    }
  }

  return (
    <KeyHint
      hint={hint}
      copyLabel={copying ? '…' : t('keys.copy')}
      copiedLabel={t('common.copied')}
      onCopy={handleCopy}
    />
  )
}