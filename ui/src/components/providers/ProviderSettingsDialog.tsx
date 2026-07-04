import { useEffect, useMemo, useState } from 'react'
import { Dialog } from '../ui/Dialog'
import { Button } from '../ui/Button'
import { Input } from '../ui/Input'
import { Select } from '../ui/Select'
import { IconPicker } from '../ui/IconPicker'
import { useUpdateProvider } from '../../hooks/useProviders'
import type { ProviderItem } from '../../hooks/useProviders'
import { getConnectionProfile } from '../../lib/connection-profiles'
import { endpointHintKey } from '../../lib/endpoint-hints'
import { useToast } from '../../hooks/useToast'
import { useTranslation } from '../../lib/i18n'

interface ProviderSettingsDialogProps {
  open: boolean
  onClose: () => void
  provider: ProviderItem
}

export function ProviderSettingsDialog({ open, onClose, provider }: ProviderSettingsDialogProps) {
  const { t } = useTranslation()
  const { toast } = useToast()
  const updateProvider = useUpdateProvider()

  const [name, setName] = useState(provider.name)
  const [slug, setSlug] = useState(provider.slug ?? '')
  const [protocol, setProtocol] = useState(provider.protocol || 'openai')
  const [baseUrl, setBaseUrl] = useState(provider.base_url ?? '')
  const [logo, setLogo] = useState(provider.logo ?? '')
  const [contact, setContact] = useState(provider.contact_info)
  const [notes, setNotes] = useState(provider.notes)
  const [status, setStatus] = useState(provider.status)
  const [rpmLimit, setRpmLimit] = useState(
    provider.rpm_limit != null && provider.rpm_limit > 0 ? String(provider.rpm_limit) : '',
  )

  useEffect(() => {
    if (!open) return
    setName(provider.name)
    setSlug(provider.slug ?? '')
    setProtocol(provider.protocol || 'openai')
    setBaseUrl(provider.base_url ?? '')
    setLogo(provider.logo ?? '')
    setContact(provider.contact_info)
    setNotes(provider.notes)
    setStatus(provider.status)
    setRpmLimit(provider.rpm_limit != null && provider.rpm_limit > 0 ? String(provider.rpm_limit) : '')
  }, [open, provider])

  const PROTOCOL_OPTIONS = useMemo(
    () => [
      { value: 'openai', label: t('providers.protocol_openai') },
      { value: 'anthropic', label: t('providers.protocol_anthropic') },
      { value: 'gemini', label: t('providers.protocol_gemini') },
      { value: 'azure', label: t('providers.protocol_azure') },
      { value: 'vertex', label: t('providers.protocol_vertex') },
      { value: 'vllm', label: t('providers.protocol_vllm') },
      { value: 'custom', label: t('providers.protocol_custom') },
    ],
    [t],
  )

  const profile = getConnectionProfile(protocol)

  function submit() {
    if (!name.trim()) {
      toast({ variant: 'error', message: t('marketplace.provider_name_required') })
      return
    }
    const rpmParsed = rpmLimit.trim() === '' ? 0 : Number.parseInt(rpmLimit, 10)
    if (rpmLimit.trim() !== '' && (Number.isNaN(rpmParsed) || rpmParsed < 0)) {
      toast({ variant: 'error', message: t('common.invalid_number') })
      return
    }
    updateProvider.mutate(
      {
        id: provider.id,
        name: name.trim(),
        slug: slug.trim() || undefined,
        protocol,
        base_url: baseUrl.trim() || undefined,
        logo: logo.trim() || undefined,
        contact_info: contact,
        notes,
        status,
        rpm_limit: rpmParsed,
      },
      {
        onSuccess: () => {
          toast({ variant: 'success', message: t('common.saved') })
          onClose()
        },
        onError: (e) => toast({ variant: 'error', message: e.message }),
      },
    )
  }

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={t('provider_detail.edit_settings')}
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            {t('common.cancel')}
          </Button>
          <Button onClick={submit} loading={updateProvider.isPending}>
            {t('common.save')}
          </Button>
        </>
      }
    >
      <p className="text-sm text-text-secondary mb-4">{t('provider_detail.edit_settings_hint')}</p>
      <div className="space-y-4">
        <Input label={t('marketplace.provider_name')} required value={name} onChange={(e) => setName(e.target.value)} />
        <Input label={t('common.slug')} value={slug} onChange={(e) => setSlug(e.target.value)} />
        <Select label={t('common.protocol')} options={PROTOCOL_OPTIONS} value={protocol} onChange={setProtocol} />
        <Input
          label={t('common.endpoint')}
          value={baseUrl}
          onChange={(e) => setBaseUrl(e.target.value)}
          placeholder={profile.baseUrlPlaceholder}
          description={t(endpointHintKey(protocol))}
        />
        <Select
          label={t('wallet.col_status')}
          options={[
            { value: 'active', label: t('providers.status_active') },
            { value: 'paused', label: t('providers.status_paused') },
          ]}
          value={status}
          onChange={(v) => setStatus(v as 'active' | 'paused')}
        />
        <IconPicker value={logo} onChange={setLogo} />
        <Input
          label={t('providers.rpm_limit')}
          type="number"
          min={0}
          value={rpmLimit}
          onChange={(e) => setRpmLimit(e.target.value)}
          placeholder={t('providers.rpm_unlimited')}
          description={t('providers.rpm_limit_desc')}
        />
        <Input label={t('marketplace.provider_contact')} value={contact} onChange={(e) => setContact(e.target.value)} />
        <Input label={t('marketplace.provider_notes')} value={notes} onChange={(e) => setNotes(e.target.value)} />
      </div>
    </Dialog>
  )
}