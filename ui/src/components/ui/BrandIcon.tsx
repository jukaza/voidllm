import { getLobeIcon } from '../../lib/lobe-icon'
import { resolveModelIcon, resolveProviderIcon } from '../../lib/provider-icons'

interface BrandIconProps {
  logo?: string | null
  slug?: string | null
  protocol?: string | null
  modelName?: string | null
  size?: number
  className?: string
}

/** Provider or model brand mark with @lobehub/icons fallback. */
export function BrandIcon({
  logo,
  slug,
  protocol,
  modelName,
  size = 20,
  className,
}: BrandIconProps) {
  const key = modelName
    ? resolveModelIcon(logo, modelName)
    : resolveProviderIcon(logo, slug, protocol)

  return (
    <span className={className} style={{ display: 'inline-flex', lineHeight: 0 }}>
      {getLobeIcon(key, size)}
    </span>
  )
}