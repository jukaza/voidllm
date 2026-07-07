import React, { useCallback, useEffect, useId, useRef, useState } from 'react'
import ReactDOM from 'react-dom'
import { getPortalRoot } from '../../lib/portalRoot'
import { cn } from '../../lib/utils'

interface CatalogTooltipProps {
  content: React.ReactNode
  children: React.ReactNode
  className?: string
}

/** Hover tooltip for catalog capability icons and stat cells. */
export function CatalogTooltip({ content, children, className }: CatalogTooltipProps) {
  const id = useId()
  const anchorRef = useRef<HTMLSpanElement>(null)
  const panelRef = useRef<HTMLDivElement>(null)
  const [open, setOpen] = useState(false)
  const [panelStyle, setPanelStyle] = useState<React.CSSProperties>({})

  const updatePosition = useCallback(() => {
    const anchor = anchorRef.current
    if (!anchor) return
    const rect = anchor.getBoundingClientRect()
    const panelWidth = 220
    let left = rect.left + rect.width / 2 - panelWidth / 2
    left = Math.max(8, Math.min(left, window.innerWidth - panelWidth - 8))
    const spaceBelow = window.innerHeight - rect.bottom
    const openAbove = spaceBelow < 100 && rect.top > 100
    setPanelStyle({
      position: 'fixed',
      left,
      width: panelWidth,
      zIndex: 260,
      ...(openAbove
        ? { bottom: window.innerHeight - rect.top + 6 }
        : { top: rect.bottom + 6 }),
    })
  }, [])

  useEffect(() => {
    if (!open) return
    updatePosition()
    const onReposition = () => updatePosition()
    window.addEventListener('resize', onReposition)
    window.addEventListener('scroll', onReposition, true)
    return () => {
      window.removeEventListener('resize', onReposition)
      window.removeEventListener('scroll', onReposition, true)
    }
  }, [open, updatePosition])

  return (
    <>
      <span
        ref={anchorRef}
        className="inline-flex"
        onMouseEnter={() => setOpen(true)}
        onMouseLeave={() => setOpen(false)}
        onFocus={() => setOpen(true)}
        onBlur={() => setOpen(false)}
        aria-describedby={open ? id : undefined}
      >
        {children}
      </span>
      {open &&
        ReactDOM.createPortal(
          <div
            ref={panelRef}
            id={id}
            role="tooltip"
            style={panelStyle}
            className={cn(
              'pointer-events-none rounded-lg border border-white/10 bg-bg-secondary px-2.5 py-2 text-[11px] leading-snug text-text-secondary shadow-xl shadow-black/50',
              className,
            )}
          >
            {content}
          </div>,
          getPortalRoot(),
        )}
    </>
  )
}