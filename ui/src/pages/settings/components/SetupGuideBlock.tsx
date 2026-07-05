import type { ReactNode } from 'react'

interface SetupGuideBlockProps {
  title: string
  description?: string
  steps: string[]
  docs?: ReactNode
  note?: string
}

export function SetupGuideBlock({ title, description, steps, docs, note }: SetupGuideBlockProps) {
  return (
    <div className="rounded-lg border border-border bg-bg-secondary">
      <div className="border-b border-border px-6 py-4">
        <h2 className="text-sm font-semibold text-text-primary">{title}</h2>
        {description && <p className="mt-1 text-xs text-text-tertiary">{description}</p>}
      </div>
      <div className="space-y-4 p-6">
        <ol className="list-decimal list-inside space-y-2 text-sm text-text-secondary">
          {steps.map((step) => (
            <li key={step}>{step}</li>
          ))}
        </ol>
        {docs && (
          <div className="rounded-md border border-border bg-bg-tertiary px-3 py-2 text-xs text-text-tertiary space-y-2">
            {docs}
          </div>
        )}
        {note && <p className="text-xs text-text-secondary">{note}</p>}
      </div>
    </div>
  )
}