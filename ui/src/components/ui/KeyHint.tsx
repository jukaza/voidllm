import React, { useEffect, useState } from 'react'
import { cn } from '../../lib/utils'

export interface KeyHintProps extends React.HTMLAttributes<HTMLDivElement> {
  /** The pre-formatted key hint from the backend (e.g. "sk-a3f...2ad6"). */
  hint: string
  /** When set, shows a copy button that copies this value (full key or hint). */
  copyValue?: string
  copyLabel?: string
  copiedLabel?: string
}

export function KeyHint({
  hint,
  copyValue,
  copyLabel = 'Copy',
  copiedLabel = 'Copied!',
  className,
  ...rest
}: KeyHintProps) {
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    if (!copied) return
    const timer = setTimeout(() => setCopied(false), 2000)
    return () => clearTimeout(timer)
  }, [copied])

  const ellipsisIdx = hint.indexOf('...')
  const prefix = ellipsisIdx >= 0 ? hint.slice(0, ellipsisIdx + 3) : ''
  const suffix = ellipsisIdx >= 0 ? hint.slice(ellipsisIdx + 3) : hint
  const canCopy = Boolean(copyValue)

  function handleCopy() {
    if (!copyValue) return
    navigator.clipboard.writeText(copyValue).then(() => setCopied(true)).catch(() => undefined)
  }

  return (
    <div className={cn('inline-flex items-center gap-1', className)} {...rest}>
      <span className="font-mono text-xs text-text-secondary">
        <span className="text-text-tertiary">{prefix}</span>
        {suffix}
      </span>
      {canCopy && (
        <button
          type="button"
          onClick={handleCopy}
          aria-label={copied ? copiedLabel : copyLabel}
          title={copied ? copiedLabel : copyLabel}
          className="shrink-0 rounded-md p-1 text-text-tertiary transition-colors hover:bg-bg-tertiary hover:text-text-primary"
        >
          {copied ? (
            <svg
              className="h-3.5 w-3.5 text-success"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.5"
              strokeLinecap="round"
              strokeLinejoin="round"
              aria-hidden="true"
            >
              <polyline points="20 6 9 17 4 12" />
            </svg>
          ) : (
            <svg
              className="h-3.5 w-3.5"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.5"
              strokeLinecap="round"
              strokeLinejoin="round"
              aria-hidden="true"
            >
              <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
              <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
            </svg>
          )}
        </button>
      )}
    </div>
  )
}