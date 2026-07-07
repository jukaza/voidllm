import { useCallback, useEffect, useId, useRef, useState } from 'react'
import ReactDOM from 'react-dom'
import { getPortalRoot } from '../../lib/portalRoot'
import { cn } from '../../lib/utils'

interface ColumnHintProps {
  text: string
  label: string
}

export function ColumnHint({ text, label }: ColumnHintProps) {
  const id = useId()
  const rootRef = useRef<HTMLDivElement>(null)
  const buttonRef = useRef<HTMLButtonElement>(null)
  const panelRef = useRef<HTMLDivElement>(null)
  const [open, setOpen] = useState(false)
  const [panelStyle, setPanelStyle] = useState<React.CSSProperties>({})

  const updatePosition = useCallback(() => {
    const btn = buttonRef.current
    if (!btn) return
    const rect = btn.getBoundingClientRect()
    const panelWidth = 240
    let left = rect.left + rect.width / 2 - panelWidth / 2
    left = Math.max(8, Math.min(left, window.innerWidth - panelWidth - 8))
    const spaceBelow = window.innerHeight - rect.bottom
    const openAbove = spaceBelow < 120 && rect.top > 120
    setPanelStyle({
      position: 'fixed',
      left,
      width: panelWidth,
      zIndex: 250,
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

  useEffect(() => {
    if (!open) return
    const onPointerDown = (e: MouseEvent) => {
      const target = e.target as Node
      if (!rootRef.current?.contains(target) && !panelRef.current?.contains(target)) {
        setOpen(false)
      }
    }
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onPointerDown)
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('mousedown', onPointerDown)
      document.removeEventListener('keydown', onKeyDown)
    }
  }, [open])

  return (
    <div ref={rootRef} className="relative inline-flex shrink-0">
      <button
        ref={buttonRef}
        type="button"
        id={id}
        onClick={(e) => {
          e.stopPropagation()
          setOpen((v) => !v)
        }}
        onKeyDown={(e) => e.stopPropagation()}
        className={cn(
          'inline-flex h-4 w-4 items-center justify-center rounded-full',
          'text-[10px] font-bold leading-none text-text-tertiary/80',
          'transition-colors hover:bg-white/10 hover:text-text-secondary',
          open && 'bg-white/10 text-text-secondary',
        )}
        aria-label={label}
        aria-expanded={open}
        aria-controls={`${id}-panel`}
      >
        ?
      </button>

      {open &&
        ReactDOM.createPortal(
          <div
            ref={panelRef}
            id={`${id}-panel`}
            role="tooltip"
            style={panelStyle}
            className="popover-enter rounded-lg border border-white/10 bg-bg-secondary px-3 py-2 text-[11px] normal-case font-normal leading-relaxed tracking-normal text-text-secondary shadow-xl shadow-black/40"
          >
            {text}
          </div>,
          getPortalRoot(),
        )}
    </div>
  )
}