import React, { useEffect, useId } from 'react'
import { cn } from '../../lib/utils'

export interface DrawerProps {
  open: boolean
  onClose: () => void
  title: string
  children: React.ReactNode
  footer?: React.ReactNode
  className?: string
  closeOnEscape?: boolean
  closeOnBackdrop?: boolean
}

/** Right-side panel — one scrollable form surface (new-api channel drawer style). */
export function Drawer({
  open,
  onClose,
  title,
  children,
  footer,
  className,
  closeOnEscape = true,
  closeOnBackdrop = true,
}: DrawerProps) {
  const titleId = useId()

  useEffect(() => {
    if (!open) return
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !e.defaultPrevented && closeOnEscape) {
        e.preventDefault()
        onClose()
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [open, onClose, closeOnEscape])

  useEffect(() => {
    if (!open) return
    const prev = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.body.style.overflow = prev
    }
  }, [open])

  if (!open) return null

  const handleBackdropMouseDown = (e: React.MouseEvent<HTMLDivElement>) => {
    if (closeOnBackdrop && e.target === e.currentTarget) onClose()
  }

  return (
    <div
      className="fixed inset-0 z-50 flex justify-end bg-black/50 backdrop-blur-sm"
      onMouseDown={handleBackdropMouseDown}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className={cn(
          'flex h-full w-full max-w-2xl flex-col border-l border-border shadow-2xl',
          'bg-bg-primary',
          className,
        )}
        onMouseDown={(e) => e.stopPropagation()}
      >
        <div className="flex shrink-0 items-center justify-between border-b border-border px-6 py-4">
          <h2 id={titleId} className="text-lg font-semibold text-text-primary">
            {title}
          </h2>
          <button
            type="button"
            onClick={onClose}
            className="text-text-tertiary hover:text-text-primary transition-colors cursor-pointer"
            aria-label="Close"
          >
            <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div className="void-scroll flex-1 overflow-y-auto px-6 py-5">
          {children}
        </div>

        {footer != null && (
          <div className="shrink-0 border-t border-border px-6 py-4">{footer}</div>
        )}
      </div>
    </div>
  )
}

export function DrawerSection({
  title,
  description,
  children,
  actions,
}: {
  title: string
  description?: string
  children: React.ReactNode
  actions?: React.ReactNode
}) {
  return (
    <section className="border-b border-border/60 pb-6 mb-6 last:border-b-0 last:mb-0 last:pb-0">
      <div className="mb-4 flex items-start justify-between gap-3">
        <div>
          <h3 className="text-xs font-medium uppercase tracking-wider text-text-tertiary">{title}</h3>
          {description ? (
            <p className="text-xs text-text-tertiary mt-1">{description}</p>
          ) : null}
        </div>
        {actions ? <div className="flex shrink-0 gap-2">{actions}</div> : null}
      </div>
      {children}
    </section>
  )
}