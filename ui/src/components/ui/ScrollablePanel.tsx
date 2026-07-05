import { useCallback, useEffect, useRef, useState } from 'react'
import { cn } from '../../lib/utils'

interface ScrollablePanelProps {
  children: React.ReactNode
  className?: string
  /** Soft gradient hints when content overflows (default: true). */
  fade?: boolean
  fadeColorClass?: string
}

export function ScrollablePanel({
  children,
  className,
  fade = true,
  fadeColorClass = 'from-bg-secondary',
}: ScrollablePanelProps) {
  const ref = useRef<HTMLDivElement>(null)
  const [fadeTop, setFadeTop] = useState(false)
  const [fadeBottom, setFadeBottom] = useState(false)

  const sync = useCallback(() => {
    const el = ref.current
    if (!el) return
    const { scrollTop, clientHeight, scrollHeight } = el
    setFadeTop(scrollTop > 6)
    setFadeBottom(scrollTop + clientHeight < scrollHeight - 6)
  }, [])

  useEffect(() => {
    sync()
    const el = ref.current
    if (!el) return
    const ro = new ResizeObserver(sync)
    ro.observe(el)
    return () => ro.disconnect()
  }, [children, sync])

  return (
    <div className="relative min-h-0">
      {fade && fadeTop && (
        <div
          className={cn(
            'pointer-events-none absolute inset-x-0 top-0 z-10 h-5 bg-gradient-to-b to-transparent',
            fadeColorClass,
          )}
          aria-hidden="true"
        />
      )}
      <div
        ref={ref}
        onScroll={sync}
        className={cn('void-scroll overflow-y-auto overflow-x-hidden', className)}
      >
        {children}
      </div>
      {fade && fadeBottom && (
        <div
          className={cn(
            'pointer-events-none absolute inset-x-0 bottom-0 z-10 h-5 bg-gradient-to-t to-transparent',
            fadeColorClass,
          )}
          aria-hidden="true"
        />
      )}
    </div>
  )
}