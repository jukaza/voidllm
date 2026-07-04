import { Button } from '../../../components/ui/Button'
import { useUsageLogDetail } from '../../../hooks/useUsageLogs'
import { useMe } from '../../../hooks/useMe'
import { LatencyBadge } from '../../../components/ui/LatencyBadge'
import { TokenBreakdown } from '../../../components/ui/TokenBreakdown'
import { formatCost, formatDate } from '../../../lib/utils'
import { useTranslation } from '../../../lib/i18n'

interface UsageLogDrawerProps {
  requestId: string | null
  onClose: () => void
}

export function UsageLogDrawer({ requestId, onClose }: UsageLogDrawerProps) {
  const { data: log, isLoading } = useUsageLogDetail(requestId)
  const { data: me } = useMe()
  const { t } = useTranslation()
  const isAdmin = me?.is_system_admin ?? false

  if (!requestId) return null

  return (
    <div className="fixed inset-0 z-50 flex justify-end">
      <button
        type="button"
        className="absolute inset-0 bg-black/40"
        aria-label="Close"
        onClick={onClose}
      />
      <aside className="relative w-full max-w-md h-full bg-bg-secondary border-l border-border p-6 overflow-y-auto shadow-xl">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-lg font-semibold text-text-primary">{t('analytics.log_detail')}</h2>
          <Button variant="ghost" size="sm" onClick={onClose}>
            {t('common.close')}
          </Button>
        </div>

        {isLoading || !log ? (
          <p className="text-text-secondary text-sm">{t('common.loading')}</p>
        ) : (
          <dl className="space-y-4 text-sm">
            <div>
              <dt className="text-text-tertiary">{t('analytics.request_id')}</dt>
              <dd className="font-mono text-text-primary break-all">{log.request_id}</dd>
            </div>
            <div>
              <dt className="text-text-tertiary">{t('analytics.time')}</dt>
              <dd className="text-text-primary">{formatDate(log.created_at)}</dd>
            </div>
            <div>
              <dt className="text-text-tertiary">Model</dt>
              <dd className="text-text-primary">{log.model_name}</dd>
            </div>
            {isAdmin && (log.provider_name || log.channel_label || log.deployment_id) && (
              <div>
                <dt className="text-text-tertiary">{t('analytics.channel')}</dt>
                <dd className="text-text-primary">
                  {log.provider_name || log.channel_label || log.deployment_id}
                  {log.channel_label && log.provider_name && log.channel_label !== log.provider_name && (
                    <span className="block text-xs text-text-tertiary mt-0.5">{log.channel_label}</span>
                  )}
                </dd>
              </div>
            )}
            <div>
              <dt className="text-text-tertiary">Tokens</dt>
              <dd>
                <TokenBreakdown
                  promptTokens={log.prompt_tokens}
                  completionTokens={log.completion_tokens}
                  cachedTokens={log.cached_tokens}
                  cacheWriteTokens={log.cache_write_tokens}
                />
              </dd>
            </div>
            <div>
              <dt className="text-text-tertiary">{t('analytics.latency')}</dt>
              <dd>
                <LatencyBadge
                  durationMs={log.duration_ms}
                  ttftMs={log.ttft_ms}
                  isStream={log.is_stream}
                  className="items-start"
                />
              </dd>
            </div>
            <div>
              <dt className="text-text-tertiary">Status</dt>
              <dd className="text-text-primary">{log.status_code}</dd>
            </div>

            {log.revenue != null && (
              <div>
                <dt className="text-text-tertiary">
                  {isAdmin ? t('analytics.revenue') : t('analytics.payment')}
                </dt>
                <dd className="text-text-primary font-medium">{formatCost(log.revenue)}</dd>
              </div>
            )}


          </dl>
        )}
      </aside>
    </div>
  )
}