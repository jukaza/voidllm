import { cn } from '../../lib/utils'
import type { PublicSubscriptionPackage } from '../../hooks/useSubscriptions'

export const presetGradients: Record<string, string> = {
  aurora: 'from-violet-600/90 via-fuchsia-500/70 to-cyan-400/80',
  sunset: 'from-orange-500/90 via-rose-500/80 to-amber-400/70',
  ocean: 'from-blue-700/90 via-cyan-500/70 to-teal-400/80',
  ember: 'from-red-700/90 via-orange-600/80 to-yellow-500/60',
  violet: 'from-indigo-700/90 via-violet-600/80 to-purple-400/70',
}

export function PackageCover({
  pkg,
  className,
  compact,
}: {
  pkg: Pick<PublicSubscriptionPackage, 'cover_type' | 'cover_value'>
  className?: string
  compact?: boolean
}) {
  const h = compact ? 'h-36' : 'h-44'
  const gradient =
    pkg.cover_type === 'default'
      ? presetGradients[pkg.cover_value] ?? presetGradients.aurora
      : null

  if (pkg.cover_type === 'upload' || pkg.cover_type === 'url') {
    const src = pkg.cover_value
    return (
      <div className={cn('relative overflow-hidden', h, className)}>
        {src ? (
          <img src={src} alt="" className="h-full w-full object-cover" />
        ) : (
          <div className={cn('h-full w-full bg-gradient-to-br', presetGradients.aurora)} />
        )}
        <div className="absolute inset-0 bg-gradient-to-t from-bg-secondary/90 via-bg-secondary/10 to-transparent" />
      </div>
    )
  }

  return (
    <div className={cn('relative overflow-hidden bg-gradient-to-br', h, gradient, className)}>
      <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top_right,rgba(255,255,255,0.15),transparent_55%)]" />
      <div className="absolute inset-0 bg-gradient-to-t from-bg-secondary/80 via-transparent to-transparent" />
    </div>
  )
}