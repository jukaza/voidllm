export interface PillOption<T extends string | number> {
  value: T
  label: string
}

export function PillGroup<T extends string | number>({
  options,
  value,
  onChange,
  label,
}: {
  options: PillOption<T>[]
  value: T
  onChange: (v: T) => void
  label?: string
}) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      {label && <span className="text-xs text-text-tertiary">{label}</span>}
      <div className="flex items-center gap-1">
        {options.map((opt) => (
          <button
            key={String(opt.value)}
            type="button"
            onClick={() => onChange(opt.value)}
            className={
              value === opt.value
                ? 'px-3 py-1 rounded-md text-xs font-medium bg-accent/20 text-accent border border-accent/30'
                : 'px-3 py-1 rounded-md text-xs font-medium text-text-tertiary hover:text-text-secondary hover:bg-bg-tertiary border border-transparent transition-colors'
            }
          >
            {opt.label}
          </button>
        ))}
      </div>
    </div>
  )
}