import { useEffect, useRef, useState } from 'react'
import { Markdown } from '../ui/Markdown'
import { useSiteConfig } from '../../hooks/useSiteConfig'
import { useNoticeRead } from '../../hooks/useNoticeRead'
import { useTranslation } from '../../lib/i18n'
import { announcementDotClass, formatAnnouncementDate } from '../../lib/announcements'
import { ScrollablePanel } from '../ui/ScrollablePanel'
import { cn } from '../../lib/utils'

function BellIcon({ className }: { className?: string }) {
  return (
    <svg
      className={className}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.75}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M18 8a6 6 0 1 0-12 0c0 7-3 9-3 9h18s-3-2-3-9" />
      <path d="M13.73 21a2 2 0 0 1-3.46 0" />
    </svg>
  )
}

interface SiteNoticeBellProps {
  className?: string
  panelAlign?: 'left' | 'right'
}

export function SiteNoticeBell({ className, panelAlign = 'right' }: SiteNoticeBellProps) {
  const { data } = useSiteConfig()
  const { t, language } = useTranslation()
  const [open, setOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)

  const announcements = data?.announcements ?? []
  const enabled = data?.notice_enabled && announcements.length > 0
  const { isUnread, unreadCount, markRead } = useNoticeRead(announcements)

  useEffect(() => {
    if (!open) return
    const onPointerDown = (e: MouseEvent) => {
      if (!rootRef.current?.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', onPointerDown)
    return () => document.removeEventListener('mousedown', onPointerDown)
  }, [open])

  useEffect(() => {
    if (!open) return
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('keydown', onKeyDown)
    return () => document.removeEventListener('keydown', onKeyDown)
  }, [open])

  if (!enabled) return null

  function toggleOpen() {
    setOpen((prev) => {
      const next = !prev
      if (next) markRead()
      return next
    })
  }

  return (
    <div ref={rootRef} className={cn('relative', className)}>
      <button
        type="button"
        onClick={toggleOpen}
        aria-label={t('notice.bell_label')}
        aria-expanded={open}
        aria-haspopup="dialog"
        className={cn(
          'relative flex h-9 w-9 items-center justify-center rounded-lg',
          'text-text-secondary transition-colors hover:bg-white/5 hover:text-text-primary',
          open && 'bg-white/5 text-text-primary',
        )}
      >
        <BellIcon className="h-[1.2rem] w-[1.2rem]" />
        {isUnread && (
          <span className="absolute -right-0.5 -top-0.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-error px-1 text-[10px] font-bold text-white">
            {unreadCount > 99 ? '99+' : unreadCount}
          </span>
        )}
      </button>

      {open && (
        <div
          role="dialog"
          aria-label={t('notice.title')}
          className={cn(
            'popover-enter absolute top-full z-50 mt-2.5 w-[min(26rem,calc(100vw-2rem))]',
            'overflow-hidden rounded-xl border border-white/10 bg-bg-secondary/95 shadow-2xl shadow-black/50 backdrop-blur-md',
            panelAlign === 'right' ? 'right-0' : 'left-0',
          )}
        >
          <div
            className={cn(
              'pointer-events-none absolute -top-1.5 h-3 w-3 rotate-45 border border-white/10 bg-bg-secondary/95',
              panelAlign === 'right' ? 'right-4' : 'left-4',
            )}
            aria-hidden="true"
          />

          <div className="border-b border-white/5 bg-white/[0.02] px-4 py-3">
            <p className="text-sm font-semibold text-text-primary">{t('notice.title')}</p>
            <p className="mt-0.5 text-xs text-text-tertiary">{t('notice.subtitle')}</p>
          </div>

          <ScrollablePanel
            className="max-h-[min(52vh,28rem)] px-4 py-2"
            fadeColorClass="from-bg-secondary/95"
          >
            {announcements.map((item, idx) => (
              <div key={item.id}>
                <div className="py-3">
                  <div className="flex items-start gap-3">
                    <span
                      className={cn(
                        'mt-1.5 inline-block h-2 w-2 shrink-0 rounded-full',
                        announcementDotClass(item.type),
                      )}
                      aria-hidden="true"
                    />
                    <div className="min-w-0 flex-1">
                      <Markdown>{item.content}</Markdown>
                      {item.extra?.trim() && (
                        <div className="mt-2 text-xs text-text-tertiary">
                          <Markdown>{item.extra}</Markdown>
                        </div>
                      )}
                      <p className="mt-2 text-[11px] text-text-tertiary">
                        {formatAnnouncementDate(item.publish_date, language)}
                      </p>
                    </div>
                  </div>
                </div>
                {idx < announcements.length - 1 && <div className="h-px bg-white/5" />}
              </div>
            ))}
          </ScrollablePanel>
        </div>
      )}
    </div>
  )
}