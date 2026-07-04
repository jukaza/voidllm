import { useMemo, useState } from 'react'
import { Dialog } from '../ui/Dialog'
import { Input } from '../ui/Input'
import { useAllUpstreamModels } from '../../hooks/useUpstreamModels'
import type { UpstreamModelItem } from '../../hooks/useUpstreamModels'
import { useTranslation } from '../../lib/i18n'
import { cn } from '../../lib/utils'
import {
  groupUpstreamModels,
  stepKey,
  upstreamToStep,
} from './upstream-model-utils'
import type { RouteStepDraft } from './ComboRouteEditor'

interface UpstreamModelSelectModalProps {
  open: boolean
  onClose: () => void
  addedSteps: RouteStepDraft[]
  onAdd: (step: RouteStepDraft) => void
  onRemove: (key: string) => void
}

export function UpstreamModelSelectModal({
  open,
  onClose,
  addedSteps,
  onAdd,
  onRemove,
}: UpstreamModelSelectModalProps) {
  const { t } = useTranslation()
  const { data, isLoading } = useAllUpstreamModels(true, open)
  const [search, setSearch] = useState('')

  const addedKeys = useMemo(() => new Set(addedSteps.map((s) => stepKey(s.provider_id, s.upstream_model))), [addedSteps])

  const upstreamModels = data?.data ?? []

  const filteredGroups = useMemo(() => {
    const q = search.trim().toLowerCase()
    const groups = groupUpstreamModels(upstreamModels)

    if (!q) return groups

    return groups
      .map((group) => {
        const providerMatches = group.providerName.toLowerCase().includes(q)
        const models = group.models.filter(
          (m) =>
            providerMatches ||
            m.upstream_id.toLowerCase().includes(q) ||
            (m.display_name?.toLowerCase().includes(q) ?? false),
        )
        if (models.length === 0) return null
        return { ...group, models }
      })
      .filter((g): g is NonNullable<typeof g> => g != null)
  }, [upstreamModels, search])

  function toggleModel(item: UpstreamModelItem) {
    const key = stepKey(item.provider_id, item.upstream_id)
    if (addedKeys.has(key)) {
      onRemove(key)
    } else {
      onAdd(upstreamToStep(item))
    }
  }

  function handleClose() {
    setSearch('')
    onClose()
  }

  return (
    <Dialog
      open={open}
      onClose={handleClose}
      title={t('models.add_upstream_models')}
      panelClassName="max-w-lg"
      stackLevel={1}
      closeOnBackdrop={false}
    >
      <div className="space-y-3 -mt-1">
        <div className="flex items-start gap-2 px-2.5 py-2 bg-accent/8 border border-accent/20 rounded-lg text-xs text-text-secondary">
          <span className="text-accent shrink-0 mt-0.5">ℹ</span>
          <span>{t('models.picker_hint')}</span>
        </div>

        <Input
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder={t('models.picker_search')}
        />

        <div className="max-h-[360px] overflow-y-auto space-y-3 pr-0.5">
          {isLoading ? (
            <p className="text-sm text-text-tertiary py-6 text-center">{t('common.loading')}</p>
          ) : filteredGroups.length === 0 ? (
            <p className="text-sm text-text-tertiary py-6 text-center">{t('models.picker_empty')}</p>
          ) : (
            filteredGroups.map((group) => (
              <div key={group.providerId}>
                <div className="flex items-center gap-1.5 mb-1.5 sticky top-0 bg-bg-primary/95 backdrop-blur-sm py-0.5 z-10">
                  <span className="text-xs font-medium text-accent">{group.providerName}</span>
                  <span className="text-[10px] text-text-tertiary">({group.models.length})</span>
                </div>
                <div className="flex flex-wrap gap-1.5">
                  {group.models.map((m) => {
                    const key = stepKey(m.provider_id, m.upstream_id)
                    const selected = addedKeys.has(key)
                    return (
                      <button
                        key={key}
                        type="button"
                        onClick={() => toggleModel(m)}
                        className={cn(
                          'px-2.5 py-1 rounded-full text-xs font-medium transition-all border flex items-center gap-1',
                          selected
                            ? 'bg-accent text-white border-accent'
                            : 'bg-bg-secondary border-border text-text-secondary hover:border-accent/50 hover:bg-accent/5',
                        )}
                      >
                        {selected ? (
                          <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" aria-hidden="true">
                            <polyline points="20 6 9 17 4 12" />
                          </svg>
                        ) : null}
                        <span className="font-mono">{m.upstream_id}</span>
                      </button>
                    )
                  })}
                </div>
              </div>
            ))
          )}
        </div>

        <div className="flex justify-end pt-1">
          <button
            type="button"
            onClick={handleClose}
            className="text-sm text-accent hover:text-accent/80 font-medium px-3 py-1.5"
          >
            {t('common.done')}
          </button>
        </div>
      </div>
    </Dialog>
  )
}