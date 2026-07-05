import { useMemo, useState } from 'react'
import { PageHeader } from '../components/ui/PageHeader'
import { Table } from '../components/ui/Table'
import type { Column } from '../components/ui/Table'
import { ConfirmDialog } from '../components/ui/Dialog'
import { Badge } from '../components/ui/Badge'
import { BillingModeBadge } from '../components/ui/BillingModeBadge'
import { Button } from '../components/ui/Button'
import { Toggle } from '../components/ui/Toggle'
import { StatCard } from '../components/ui/StatCard'
import { BrandIcon } from '../components/ui/BrandIcon'
import { CreateProductDialog } from '../components/models/CreateProductDialog'
import { EditProductDialog } from '../components/models/EditProductDialog'
import { ROUTING_STRATEGY_OPTIONS } from '../components/models/ComboRouteEditor'
import { Select } from '../components/ui/Select'
import { billingModeFromModel, useModels, useDeleteModel, useToggleModel, useUpdateModel } from '../hooks/useModels'
import type { ModelResponse } from '../hooks/useModels'
import { useToast } from '../hooks/useToast'
import { useTranslation } from '../lib/i18n'
import { formatCost } from '../lib/utils'
import { PriceCell } from '../components/ui/PriceDisplay'

const typeLabels: Record<string, string> = {
  chat: 'Chat',
  embedding: 'Embedding',
  reranking: 'Reranking',
  completion: 'Completion',
  image: 'Image',
  audio_transcription: 'Audio',
  tts: 'TTS',
}

const typeBadgeVariant: Record<string, 'default' | 'info' | 'muted' | 'success' | 'warning'> = {
  chat: 'default',
  embedding: 'info',
  reranking: 'info',
  completion: 'muted',
  image: 'success',
  audio_transcription: 'warning',
  tts: 'warning',
}

function IconLayers() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M12 2 2 7l10 5 10-5-10-5z" />
      <path d="M2 17l10 5 10-5" />
      <path d="M2 12l10 5 10-5" />
    </svg>
  )
}

function IconActivity() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <polyline points="22 12 18 12 15 21 9 3 6 12 2 12" />
    </svg>
  )
}

function IconPauseCircle() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="12" cy="12" r="10" />
      <line x1="10" y1="15" x2="10" y2="9" />
      <line x1="14" y1="15" x2="14" y2="9" />
    </svg>
  )
}

function IconPencil() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
      <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
    </svg>
  )
}

function IconTrash() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <polyline points="3 6 5 6 21 6" />
      <path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6" />
      <path d="M10 11v6" />
      <path d="M14 11v6" />
      <path d="M9 6V4a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2" />
    </svg>
  )
}

function SellPriceCell({ row }: { row: ModelResponse }) {
  if (billingModeFromModel(row) === 'request') {
    return (
      <div className="text-xs">
        <PriceCell value={row.sell_per_request} />
        {row.sell_per_request != null && row.sell_per_request > 0 && (
          <span className="text-text-tertiary"> /req</span>
        )}
      </div>
    )
  }

  const lines: { label: string; value: number }[] = []
  if (row.sell_input_per_1m != null && row.sell_input_per_1m > 0) {
    lines.push({ label: 'in', value: row.sell_input_per_1m })
  }
  if (row.sell_output_per_1m != null && row.sell_output_per_1m > 0) {
    lines.push({ label: 'out', value: row.sell_output_per_1m })
  }
  if (row.bill_min_per_request && row.sell_min_per_request != null && row.sell_min_per_request > 0) {
    lines.push({ label: 'min', value: row.sell_min_per_request })
  }

  if (lines.length === 0) return <span className="text-text-tertiary">—</span>

  return (
    <div className="flex flex-col gap-0.5 text-xs tabular-nums">
      {lines.map((line) => (
        <div key={line.label} className="flex items-baseline gap-1.5">
          <span className="text-[10px] uppercase tracking-wide text-text-tertiary w-8 shrink-0">
            {line.label}
          </span>
          <span className="text-text-secondary">{formatCost(line.value)}</span>
        </div>
      ))}
    </div>
  )
}

function MinPerRequestToggle({ model }: { model: ModelResponse }) {
  const updateModel = useUpdateModel()
  const { toast } = useToast()
  const { t } = useTranslation()

  if (model.source !== 'api' || billingModeFromModel(model) !== 'token') {
    return <span className="text-text-tertiary">—</span>
  }

  const hasPrice = model.sell_min_per_request != null && model.sell_min_per_request > 0
  const enabled = model.bill_min_per_request === true

  return (
    <div className="flex items-center gap-2">
      <Toggle
        checked={enabled}
        size="sm"
        disabled={
          (updateModel.isPending && updateModel.variables?.modelId === model.id) ||
          (!hasPrice && !enabled)
        }
        onChange={(on) => {
          if (on && !hasPrice) {
            toast({ variant: 'error', message: t('models.min_per_request_needs_price') })
            return
          }
          updateModel.mutate(
            { modelId: model.id, params: { bill_min_per_request: on } },
            {
              onError: (err) => {
                toast({
                  variant: 'error',
                  message: err instanceof Error ? err.message : t('models.product_update_failed'),
                })
              },
            },
          )
        }}
      />
      {hasPrice ? (
        <span className="text-xs text-text-secondary tabular-nums">{formatCost(model.sell_min_per_request!)}</span>
      ) : null}
    </div>
  )
}

function PublicToggle({ model }: { model: ModelResponse }) {
  const updateModel = useUpdateModel()
  const { toast } = useToast()
  const { t } = useTranslation()

  if (model.source !== 'api') {
    return <span className="text-text-tertiary">—</span>
  }

  return (
    <Toggle
      checked={model.is_public === true}
      size="sm"
      disabled={updateModel.isPending && updateModel.variables?.modelId === model.id}
      onChange={(on) => {
        updateModel.mutate(
          { modelId: model.id, params: { is_public: on } },
          {
            onError: (err) => {
              toast({
                variant: 'error',
                message: err instanceof Error ? err.message : t('models.product_update_failed'),
              })
            },
          },
        )
      }}
      aria-label={t('models.public_storefront')}
    />
  )
}

function StrategySelect({ modelId, value }: { modelId: string; value: string }) {
  const updateModel = useUpdateModel()
  const { toast } = useToast()
  const { t } = useTranslation()
  const current = value || 'fallback'

  const strategyOptions = useMemo(
    () =>
      ROUTING_STRATEGY_OPTIONS.map((opt) => ({
        ...opt,
        label: opt.value === 'fallback' ? t('models.strategy_fallback') : t('models.strategy_round_robin'),
        description:
          opt.value === 'fallback'
            ? t('models.strategy_fallback_desc')
            : t('models.strategy_round_robin_desc'),
      })),
    [t],
  )

  return (
    <div className="min-w-[11rem]">
      <Select
        options={strategyOptions}
        value={current}
        fullWidth={false}
        onChange={(next) => {
          if (next === current) return
          updateModel.mutate(
            { modelId, params: { routing_strategy: next } },
            {
              onError: (err) => {
                toast({
                  variant: 'error',
                  message: err instanceof Error ? err.message : t('models.strategy_update_failed'),
                })
              },
            },
          )
        }}
        disabled={updateModel.isPending && updateModel.variables?.modelId === modelId}
      />
    </div>
  )
}

function RouteCountBadge({ count }: { count: number }) {
  if (count === 0) return <Badge variant="muted">0</Badge>
  return <Badge variant="info">{count}</Badge>
}

export default function ModelsPage() {
  const { t } = useTranslation()
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [editModel, setEditModel] = useState<ModelResponse | null>(null)
  const [deleteModelId, setDeleteModelId] = useState<string | null>(null)

  const { data: models, isLoading } = useModels()
  const deleteModel = useDeleteModel()
  const toggleModel = useToggleModel()
  const { toast } = useToast()

  const allModels = models?.data ?? []
  const apiProducts = useMemo(() => allModels.filter((m) => m.source === 'api'), [allModels])
  const activeCount = useMemo(() => apiProducts.filter((m) => m.is_active).length, [apiProducts])
  const publishedCount = useMemo(
    () => apiProducts.filter((m) => m.is_active && m.is_public).length,
    [apiProducts],
  )

  const columns: Column<ModelResponse>[] = useMemo(() => [
    {
      key: 'logo',
      header: '',
      width: '36px',
      render: (row) => <BrandIcon logo={row.logo} modelName={row.name} size={22} />,
    },
    {
      key: 'name',
      header: t('models.col_name'),
      render: (row) => (
        <div className="min-w-0">
          <span className="font-mono text-text-primary text-sm">{row.name}</span>
          {row.source === 'yaml' && (
            <Badge variant="muted" className="ml-2 text-[10px]">yaml</Badge>
          )}
        </div>
      ),
    },
    {
      key: 'type',
      header: t('models.col_type'),
      render: (row) => (
        <Badge variant={typeBadgeVariant[row.type] ?? 'muted'}>
          {typeLabels[row.type] ?? row.type ?? 'Chat'}
        </Badge>
      ),
    },
    {
      key: 'billing',
      header: t('models.col_billing'),
      render: (row) => {
        if (row.source !== 'api') return <span className="text-text-tertiary">—</span>
        return <BillingModeBadge mode={billingModeFromModel(row)} />
      },
    },
    {
      key: 'sell',
      header: t('models.col_sell_price'),
      render: (row) => <SellPriceCell row={row} />,
    },
    {
      key: 'min_per_request',
      header: t('models.col_min_per_request'),
      render: (row) => <MinPerRequestToggle model={row} />,
    },
    {
      key: 'routes',
      header: t('models.col_routes'),
      render: (row) =>
        row.source === 'api' ? (
          <RouteCountBadge count={row.route_count ?? 0} />
        ) : (
          <span className="text-text-tertiary">—</span>
        ),
    },
    {
      key: 'strategy',
      header: t('models.col_strategy'),
      render: (row) =>
        row.source === 'api' ? (
          <StrategySelect modelId={row.id} value={row.routing_strategy ?? 'fallback'} />
        ) : (
          <Badge variant="muted">{row.routing_strategy || 'fallback'}</Badge>
        ),
    },
    {
      key: 'public',
      header: t('models.col_public'),
      headerHint: t('models.col_public_hint'),
      render: (row) => <PublicToggle model={row} />,
    },
    {
      key: 'aliases',
      header: t('models.col_aliases'),
      headerHint: t('models.col_aliases_hint'),
      render: (row) => {
        const list = row.aliases ?? []
        if (list.length === 0) {
          if (row.source !== 'api') {
            return <span className="text-text-tertiary">—</span>
          }
          return (
            <div className="min-w-[7.5rem]">
              <p className="text-xs text-text-tertiary">{t('models.aliases_empty')}</p>
              <p className="text-[10px] text-text-tertiary/70">{t('models.aliases_example')}</p>
              <button
                type="button"
                onClick={() => setEditModel(row)}
                className="mt-1 text-[11px] font-medium text-accent hover:opacity-80"
              >
                {t('models.aliases_add')}
              </button>
            </div>
          )
        }
        return (
          <div className="flex flex-wrap gap-1">
            {list.slice(0, 2).map((a) => (
              <Badge key={a} variant="muted">{a}</Badge>
            ))}
            {list.length > 2 && <Badge variant="muted">+{list.length - 2}</Badge>}
          </div>
        )
      },
    },
    {
      key: 'is_active',
      header: t('common.status'),
      render: (row) => (
        <Toggle
          checked={row.is_active}
          onChange={(activate) =>
            toggleModel.mutate(
              { modelId: row.id, activate },
              {
                onError: (err) => {
                  toast({
                    variant: 'error',
                    message: err instanceof Error ? err.message : t('models.toggle_failed'),
                  })
                },
              },
            )
          }
          disabled={toggleModel.isPending && toggleModel.variables?.modelId === row.id}
          size="sm"
        />
      ),
    },
    {
      key: 'actions',
      header: '',
      align: 'right',
      render: (row) => {
        if (row.source !== 'api') return null
        return (
          <div className="flex items-center justify-end gap-1">
            <button
              type="button"
              onClick={() => setEditModel(row)}
              title={t('models.edit_product')}
              className="p-1.5 rounded-md text-text-tertiary hover:text-text-primary hover:bg-bg-tertiary transition-colors"
            >
              <IconPencil />
            </button>
            <button
              type="button"
              onClick={() => setDeleteModelId(row.id)}
              disabled={deleteModel.isPending && deleteModelId === row.id}
              title={t('models.delete')}
              className="p-1.5 rounded-md text-text-tertiary hover:text-error hover:bg-error/10 transition-colors disabled:opacity-40"
            >
              <IconTrash />
            </button>
          </div>
        )
      },
    },
  ], [t, toggleModel, deleteModel, deleteModelId, toast, setEditModel])

  function handleDelete() {
    if (!deleteModelId) return
    deleteModel.mutate(deleteModelId, {
      onSuccess: () => {
        toast({ variant: 'success', message: t('models.product_deleted') })
        setDeleteModelId(null)
      },
      onError: (err) => {
        toast({
          variant: 'error',
          message: err instanceof Error ? err.message : t('models.delete_failed'),
        })
        setDeleteModelId(null)
      },
    })
  }

  return (
    <>
      <PageHeader
        title={t('models.title')}
        description={t('models.desc')}
        actions={
          <Button onClick={() => setShowCreateDialog(true)}>{t('models.add_product')}</Button>
        }
      />

      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-6">
        <StatCard
          label={t('models.total_products')}
          value={isLoading ? '—' : apiProducts.length}
          icon={<IconLayers />}
          iconColor="purple"
        />
        <StatCard
          label={t('models.active')}
          value={isLoading ? '—' : activeCount}
          icon={<IconActivity />}
          iconColor="green"
        />
        <StatCard
          label={t('models.published')}
          value={isLoading ? '—' : publishedCount}
          icon={<IconPauseCircle />}
          iconColor="yellow"
        />
      </div>

      <Table<ModelResponse>
        columns={columns}
        data={allModels}
        keyExtractor={(row) => row.id}
        loading={isLoading}
        emptyMessage={t('models.empty')}
      />

      <CreateProductDialog
        open={showCreateDialog}
        onClose={() => setShowCreateDialog(false)}
      />

      {editModel !== null && (
        <EditProductDialog model={editModel} onClose={() => setEditModel(null)} />
      )}

      <ConfirmDialog
        open={deleteModelId !== null}
        onClose={() => setDeleteModelId(null)}
        onConfirm={handleDelete}
        title={t('models.delete')}
        description={t('models.delete_confirm')}
        confirmLabel={t('common.delete')}
        loading={deleteModel.isPending}
      />
    </>
  )
}