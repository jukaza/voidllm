import { Input } from '../ui/Input'
import type { CatalogModelItem } from '../../hooks/useProviders'
import {
  type ModelFamily,
  FAMILY_FILTER_ORDER,
  FAMILY_LOBE_ICON,
  anyModelHasCache,
  detectFamily,
} from '../../lib/catalog-utils'
import { getLobeIcon } from '../../lib/lobe-icon'
import { useTranslation, type TranslationKey } from '../../lib/i18n'
import { cn } from '../../lib/utils'

const FAMILY_LABEL_KEYS: Record<ModelFamily, TranslationKey> = {
  all: 'catalog.family_all',
  claude: 'catalog.family_claude',
  gpt: 'catalog.family_gpt',
  gemini: 'catalog.family_gemini',
  deepseek: 'catalog.family_deepseek',
  qwen: 'catalog.family_qwen',
  glm: 'catalog.family_glm',
  kimi: 'catalog.family_kimi',
  minimax: 'catalog.family_minimax',
  grok: 'catalog.family_grok',
  llama: 'catalog.family_llama',
  mistral: 'catalog.family_mistral',
  cohere: 'catalog.family_cohere',
  other: 'catalog.family_other',
}

interface ModelCatalogToolbarProps {
  models: CatalogModelItem[]
  search: string
  family: ModelFamily
  cacheOnly: boolean
  shown: number
  total: number
  onSearchChange: (value: string) => void
  onFamilyChange: (family: ModelFamily) => void
  onCacheOnlyChange: (value: boolean) => void
  onOpenFilters?: () => void
}

function CacheFilterChip({
  active,
  onClick,
}: {
  active: boolean
  onClick: () => void
}) {
  const { t } = useTranslation()
  return (
    <button
      type="button"
      title={t('catalog.cache_filter')}
      aria-label={t('catalog.cache_filter')}
      aria-pressed={active}
      onClick={onClick}
      className={cn(
        'inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border transition-colors',
        active
          ? 'border-success/40 bg-success/15 text-success'
          : 'border-border bg-bg-secondary text-text-tertiary hover:border-white/10 hover:text-text-secondary',
      )}
    >
      <svg
        className="h-4 w-4"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth={2}
        strokeLinecap="round"
        strokeLinejoin="round"
        aria-hidden="true"
      >
        <path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z" />
      </svg>
    </button>
  )
}

function AllFamiliesIcon() {
  return (
    <svg
      className="h-4 w-4 shrink-0 text-current"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      aria-hidden="true"
    >
      <rect x="3" y="3" width="7" height="7" rx="1" />
      <rect x="14" y="3" width="7" height="7" rx="1" />
      <rect x="3" y="14" width="7" height="7" rx="1" />
      <rect x="14" y="14" width="7" height="7" rx="1" />
    </svg>
  )
}

function FamilyQuickChip({
  family: option,
  active,
  label,
  onSelect,
}: {
  family: ModelFamily
  active: boolean
  label: string
  onSelect: () => void
}) {
  const icon =
    option === 'all' ? (
      <AllFamiliesIcon />
    ) : (
      getLobeIcon(FAMILY_LOBE_ICON[option], 16)
    )

  return (
    <button
      type="button"
      title={label}
      aria-label={label}
      aria-pressed={active}
      onClick={onSelect}
      className={cn(
        'inline-flex h-8 shrink-0 items-center gap-1.5 rounded-lg border px-2 transition-colors',
        active
          ? 'border-accent/40 bg-accent/15 text-accent'
          : 'border-border bg-bg-secondary/60 text-text-secondary hover:border-white/10 hover:bg-bg-tertiary hover:text-text-primary',
      )}
    >
      <span className="inline-flex leading-none">{icon}</span>
      <span className="font-mono text-[11px] font-medium">{label}</span>
    </button>
  )
}

export function ModelCatalogToolbar({
  models,
  search,
  family,
  cacheOnly,
  shown,
  total,
  onSearchChange,
  onFamilyChange,
  onCacheOnlyChange,
  onOpenFilters,
}: ModelCatalogToolbarProps) {
  const { t } = useTranslation()
  const showCacheChip = anyModelHasCache(models)

  const visibleFamilies = FAMILY_FILTER_ORDER.filter((option) => {
    if (option === 'all') return true
    return models.some((m) => detectFamily(m.name) === option)
  })

  return (
    <div className="space-y-3">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex min-w-0 flex-1 items-center gap-2">
          <div className="min-w-0 flex-1 max-w-md">
            <Input
              value={search}
              onChange={(e) => onSearchChange(e.target.value)}
              placeholder={t('catalog.search_placeholder')}
              aria-label={t('catalog.search_placeholder')}
            />
          </div>
          {showCacheChip && (
            <CacheFilterChip
              active={cacheOnly}
              onClick={() => onCacheOnlyChange(!cacheOnly)}
            />
          )}
          {onOpenFilters && (
            <button
              type="button"
              onClick={onOpenFilters}
              className="lg:hidden shrink-0 rounded-lg border border-border bg-bg-secondary px-3 py-2 text-xs font-medium text-text-secondary hover:text-text-primary"
            >
              {t('catalog.filter_title')}
            </button>
          )}
        </div>
        <p className="shrink-0 font-mono text-xs text-text-tertiary">
          {t('catalog.count', { shown, total })}
        </p>
      </div>

      <div className="-mx-1 flex gap-1.5 overflow-x-auto px-1 pb-0.5 scrollbar-thin">
        {visibleFamilies.map((option) => (
          <FamilyQuickChip
            key={option}
            family={option}
            active={family === option}
            label={t(FAMILY_LABEL_KEYS[option])}
            onSelect={() => onFamilyChange(option)}
          />
        ))}
      </div>
    </div>
  )
}