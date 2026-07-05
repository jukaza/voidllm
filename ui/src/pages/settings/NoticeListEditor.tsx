import { useEffect, useRef, useState } from 'react'
import { Button } from '../../components/ui/Button'
import { Textarea } from '../../components/ui/Textarea'
import { Markdown } from '../../components/ui/Markdown'
import {
  announcementDotClass,
  createAnnouncement,
  createSampleOpeningAnnouncement,
  type AnnouncementType,
  type SiteAnnouncement,
} from '../../lib/announcements'
import { useSiteConfig } from '../../hooks/useSiteConfig'
import { useTranslation } from '../../lib/i18n'
import { cn } from '../../lib/utils'

interface NoticeListEditorProps {
  items: SiteAnnouncement[]
  onChange: (items: SiteAnnouncement[]) => void
}

function contentPreview(content: string): string {
  const line = content
    .split('\n')
    .map((s) => s.trim())
    .find(Boolean)
  if (!line) return ''
  return line.replace(/!\[[^\]]*]\([^)]+\)/g, '[ảnh]').replace(/[#*_>`[\]]/g, '').trim()
}

export function NoticeListEditor({ items, onChange }: NoticeListEditorProps) {
  const { t } = useTranslation()
  const { data: site } = useSiteConfig()
  const [activeId, setActiveId] = useState<string | null>(null)
  const [showPreview, setShowPreview] = useState(false)
  const editorRef = useRef<HTMLDivElement>(null)

  const typeOptions: { value: AnnouncementType; label: string }[] = [
    { value: 'default', label: t('notice.type_default') },
    { value: 'ongoing', label: t('notice.type_ongoing') },
    { value: 'success', label: t('notice.type_success') },
    { value: 'warning', label: t('notice.type_warning') },
    { value: 'error', label: t('notice.type_error') },
  ]

  useEffect(() => {
    if (activeId && !items.some((item) => item.id === activeId)) {
      setActiveId(null)
      setShowPreview(false)
    }
  }, [activeId, items])

  useEffect(() => {
    if (!activeId || !editorRef.current) return
    editorRef.current.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
  }, [activeId])

  function updateItem(id: string, patch: Partial<SiteAnnouncement>) {
    onChange(items.map((item) => (item.id === id ? { ...item, ...patch } : item)))
  }

  function removeItem(id: string) {
    onChange(items.filter((item) => item.id !== id))
    if (activeId === id) {
      setActiveId(null)
      setShowPreview(false)
    }
  }

  function addItem() {
    const item = createAnnouncement()
    onChange([...items, item])
    setActiveId(item.id)
    setShowPreview(false)
  }

  function insertSample() {
    const item = createSampleOpeningAnnouncement(site?.system_name ?? 'VoidLLM')
    onChange([...items, item])
    setActiveId(item.id)
    setShowPreview(true)
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-xs text-text-tertiary">{t('notice.editor_hint')}</p>
        <div className="flex flex-wrap gap-2">
          <Button type="button" size="sm" variant="secondary" onClick={insertSample}>
            {t('notice.insert_sample')}
          </Button>
          <Button type="button" size="sm" onClick={addItem}>
            {t('notice.add')}
          </Button>
        </div>
      </div>

      {items.length === 0 ? (
        <div className="rounded-lg border border-dashed border-border px-4 py-8 text-center text-sm text-text-tertiary">
          {t('notice.empty')}
        </div>
      ) : (
        <ul className="space-y-2">
          {items.map((item, index) => {
            const isActive = activeId === item.id
            const preview = contentPreview(item.content)
            const typeLabel = typeOptions.find((opt) => opt.value === item.type)?.label ?? item.type

            return (
              <li
                key={item.id}
                className={cn(
                  'rounded-lg border transition-colors',
                  isActive ? 'border-accent/40 bg-accent/5' : 'border-border bg-bg-primary/30',
                )}
              >
                <div className="flex items-center gap-3 px-3 py-2.5">
                  <span
                    className={cn('h-2 w-2 shrink-0 rounded-full', announcementDotClass(item.type))}
                    aria-hidden="true"
                  />
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm text-text-primary">
                      {preview || t('notice.draft')}
                    </p>
                    <p className="text-[11px] text-text-tertiary">
                      #{items.length - index} · {typeLabel}
                    </p>
                  </div>
                  <div className="flex shrink-0 items-center gap-1">
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={() => {
                        setActiveId(isActive ? null : item.id)
                        setShowPreview(false)
                      }}
                    >
                      {isActive ? t('notice.collapse') : t('common.edit')}
                    </Button>
                    <Button
                      type="button"
                      variant="destructive"
                      size="sm"
                      onClick={() => removeItem(item.id)}
                    >
                      {t('common.delete')}
                    </Button>
                  </div>
                </div>

                {isActive && (
                  <div ref={editorRef} className="space-y-4 border-t border-border/60 px-4 py-4">
                    <div>
                      <p className="mb-2 text-sm font-medium text-text-secondary">{t('notice.type')}</p>
                      <div className="flex flex-wrap gap-2">
                        {typeOptions.map((opt) => (
                          <button
                            key={opt.value}
                            type="button"
                            onClick={() => updateItem(item.id, { type: opt.value })}
                            className={cn(
                              'rounded-full border px-3 py-1 text-xs font-medium transition-colors',
                              item.type === opt.value
                                ? 'border-accent bg-accent/15 text-accent'
                                : 'border-border text-text-secondary hover:border-accent/40 hover:text-text-primary',
                            )}
                          >
                            {opt.label}
                          </button>
                        ))}
                      </div>
                    </div>

                    <Textarea
                      label={t('notice.content')}
                      value={item.content}
                      onChange={(e) => updateItem(item.id, { content: e.target.value })}
                      rows={6}
                      description={t('notice.content_hint')}
                      placeholder={t('notice.content_placeholder')}
                    />

                    <Textarea
                      label={t('notice.extra')}
                      value={item.extra ?? ''}
                      onChange={(e) => updateItem(item.id, { extra: e.target.value })}
                      rows={2}
                      description={t('notice.extra_hint')}
                    />

                    <div className="flex items-center justify-between gap-3">
                      <button
                        type="button"
                        onClick={() => setShowPreview((v) => !v)}
                        className="text-xs font-medium text-accent hover:opacity-80"
                      >
                        {showPreview ? t('notice.hide_preview') : t('notice.show_preview')}
                      </button>
                    </div>

                    {showPreview && (
                      <div className="rounded-lg border border-white/5 bg-bg-secondary p-4">
                        {item.content.trim() ? (
                          <Markdown>{item.content}</Markdown>
                        ) : (
                          <p className="text-sm text-text-tertiary">{t('notice.preview_empty')}</p>
                        )}
                        {item.extra?.trim() && (
                          <div className="mt-2 border-t border-white/5 pt-2 text-xs text-text-tertiary">
                            <Markdown>{item.extra}</Markdown>
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                )}
              </li>
            )
          })}
        </ul>
      )}
    </div>
  )
}