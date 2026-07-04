import { useMemo, useState } from 'react'
import { Table } from '../ui/Table'
import type { Column } from '../ui/Table'
import { Badge } from '../ui/Badge'
import { Button } from '../ui/Button'
import { Dialog, ConfirmDialog } from '../ui/Dialog'
import { Toggle } from '../ui/Toggle'
import {
  useProviderUpstreamModels,
  useImportProviderUpstreamModels,
  useUpdateProviderUpstreamModel,
  useDeleteProviderUpstreamModel,
} from '../../hooks/useUpstreamModels'
import type { UpstreamModelItem } from '../../hooks/useUpstreamModels'
import {
  useDiscoverProviderModels,
  type DiscoveredModel,
} from '../../hooks/useProviders'
import { useToast } from '../../hooks/useToast'
import { useTranslation } from '../../lib/i18n'

interface UpstreamModelsSectionProps {
  providerId: string
}

export function UpstreamModelsSection({ providerId }: UpstreamModelsSectionProps) {
  const { t } = useTranslation()
  const { toast } = useToast()
  const { data, isLoading } = useProviderUpstreamModels(providerId)
  const importModels = useImportProviderUpstreamModels(providerId)
  const updateModel = useUpdateProviderUpstreamModel(providerId)
  const deleteModel = useDeleteProviderUpstreamModel(providerId)
  const discover = useDiscoverProviderModels()

  const models = data?.data ?? []

  const [importOpen, setImportOpen] = useState(false)
  const [discovered, setDiscovered] = useState<DiscoveredModel[]>([])
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [deleting, setDeleting] = useState<UpstreamModelItem | null>(null)

  const inventoryIds = useMemo(
    () => new Set(models.map((m) => m.upstream_id)),
    [models],
  )

  async function runDiscover() {
    try {
      const res = await discover.mutateAsync({ provider_id: providerId })
      if (!res.success && res.data.length === 0) {
        toast({ variant: 'error', message: res.message || t('wizard.discover_failed') })
        return
      }
      setDiscovered(res.data)
      const newIds = res.data.filter((m) => !inventoryIds.has(m.id)).map((m) => m.id)
      setSelected(new Set(newIds.length > 0 ? newIds : res.data.map((m) => m.id)))
      setImportOpen(true)
      if (res.message) {
        toast({ variant: res.success ? 'success' : 'info', message: res.message })
      }
    } catch (e) {
      toast({
        variant: 'error',
        message: e instanceof Error ? e.message : t('wizard.discover_failed'),
      })
    }
  }

  function toggleModel(id: string) {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  function runImport() {
    if (selected.size === 0) return
    importModels.mutate(
      [...selected].map((id) => {
        const dm = discovered.find((m) => m.id === id)
        return {
          upstream_id: id,
          cost: dm?.known_cost
            ? {
                in: dm.known_cost.in,
                out: dm.known_cost.out,
                cached_in: dm.known_cost.cached_in,
                cache_write: dm.known_cost.cache_write,
              }
            : undefined,
        }
      }),
      {
        onSuccess: (res) => {
          const failed = res.results.filter((r) => r.error)
          const ok = res.results.length - failed.length
          if (failed.length > 0) {
            toast({
              variant: 'info',
              message: t('provider_detail.import_partial', { ok, failed: failed.length }),
            })
          } else {
            toast({ variant: 'success', message: t('provider_detail.import_success', { ok }) })
          }
          setImportOpen(false)
          setDiscovered([])
          setSelected(new Set())
        },
        onError: (e) => toast({ variant: 'error', message: e.message }),
      },
    )
  }

  function toggleEnabled(row: UpstreamModelItem) {
    updateModel.mutate(
      { modelId: row.id, is_enabled: !row.is_enabled },
      { onError: (e) => toast({ variant: 'error', message: e.message }) },
    )
  }

  const columns: Column<UpstreamModelItem>[] = [
    {
      key: 'upstream_id',
      header: t('wizard.col_upstream'),
      render: (row) => (
        <span className="font-mono text-xs text-text-primary">{row.upstream_id}</span>
      ),
    },
    {
      key: 'ref_cost',
      header: t('provider_detail.col_ref_cost'),
      render: (row) =>
        row.cost_input_per_1m != null && row.cost_output_per_1m != null ? (
          <span className="text-xs tabular-nums text-text-tertiary">
            ${row.cost_input_per_1m} / ${row.cost_output_per_1m}
          </span>
        ) : (
          <span className="text-text-tertiary text-xs">—</span>
        ),
    },
    {
      key: 'enabled',
      header: t('provider_detail.col_in_inventory'),
      render: (row) => (
        <div className="flex items-center gap-2">
          <Toggle
            checked={row.is_enabled}
            onChange={() => toggleEnabled(row)}
            size="sm"
          />
          <span className="text-xs text-text-tertiary">
            {row.is_enabled
              ? t('provider_detail.inventory_enabled')
              : t('provider_detail.inventory_disabled')}
          </span>
        </div>
      ),
    },
    {
      key: 'actions',
      header: '',
      align: 'right',
      render: (row) => (
        <Button size="sm" variant="destructive" onClick={() => setDeleting(row)}>
          {t('common.delete')}
        </Button>
      ),
    },
  ]

  return (
    <>
      <div className="mb-4 flex flex-wrap justify-between items-center gap-2">
        <p className="text-sm text-text-secondary max-w-xl">{t('provider_detail.models_desc')}</p>
        <Button
          size="sm"
          variant="secondary"
          onClick={() => void runDiscover()}
          loading={discover.isPending}
        >
          {t('provider_detail.import_discover')}
        </Button>
      </div>

      <Table
        columns={columns}
        data={models}
        keyExtractor={(r) => r.id}
        loading={isLoading}
        emptyMessage={t('provider_detail.models_empty')}
        compact
      />

      <Dialog
        open={importOpen}
        onClose={() => setImportOpen(false)}
        title={t('provider_detail.import_discover')}
        footer={
          <>
            <Button variant="secondary" onClick={() => setImportOpen(false)}>
              {t('common.cancel')}
            </Button>
            <Button
              onClick={runImport}
              loading={importModels.isPending}
              disabled={selected.size === 0}
            >
              {t('provider_detail.import_selected', { count: selected.size })}
            </Button>
          </>
        }
      >
        {discovered.length === 0 ? (
          <p className="text-sm text-text-tertiary">{t('provider_detail.discover_empty')}</p>
        ) : (
          <div className="rounded-lg border border-border overflow-hidden max-h-80 overflow-y-auto">
            <table className="w-full text-sm">
              <thead className="bg-bg-secondary text-text-tertiary text-left sticky top-0">
                <tr>
                  <th className="px-3 py-2 w-8" />
                  <th className="px-3 py-2">{t('wizard.col_upstream')}</th>
                  <th className="px-3 py-2 hidden sm:table-cell">{t('provider_detail.col_ref_cost')}</th>
                  <th className="px-3 py-2">{t('common.status')}</th>
                </tr>
              </thead>
              <tbody>
                {discovered.map((m) => {
                  const inInventory = inventoryIds.has(m.id)
                  return (
                    <tr key={m.id} className="border-t border-border/60">
                      <td className="px-3 py-2">
                        <input
                          type="checkbox"
                          checked={selected.has(m.id)}
                          onChange={() => toggleModel(m.id)}
                          className="accent-accent"
                        />
                      </td>
                      <td className="px-3 py-2 font-mono text-xs">{m.id}</td>
                      <td className="px-3 py-2 text-xs tabular-nums text-text-tertiary hidden sm:table-cell">
                        {m.known_cost
                          ? `$${m.known_cost.in} / $${m.known_cost.out}`
                          : '—'}
                      </td>
                      <td className="px-3 py-2">
                        {inInventory ? (
                          <Badge variant="info">{t('provider_detail.badge_in_inventory')}</Badge>
                        ) : (
                          <Badge variant="success">{t('provider_detail.badge_new')}</Badge>
                        )}
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
      </Dialog>

      <ConfirmDialog
        open={deleting !== null}
        onClose={() => setDeleting(null)}
        onConfirm={() => {
          if (!deleting) return
          deleteModel.mutate(deleting.id, {
            onSuccess: () => {
              toast({ variant: 'success', message: t('common.deleted') })
              setDeleting(null)
            },
            onError: (e) => {
              toast({ variant: 'error', message: e.message })
              setDeleting(null)
            },
          })
        }}
        title={t('provider_detail.confirm_delete_model')}
        description={deleting?.upstream_id ?? ''}
        confirmLabel={t('common.delete')}
        loading={deleteModel.isPending}
      />
    </>
  )
}