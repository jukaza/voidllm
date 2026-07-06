import { useSiteConfig } from '../../hooks/useSiteConfig'

interface BrandMarkProps {
  className?: string
  nameClassName?: string
  iconClassName?: string
  showName?: boolean
}

export function BrandMark({
  className = 'flex items-center gap-2',
  nameClassName = 'gradient-text text-xl font-bold',
  iconClassName = 'h-7 w-7',
  showName = true,
}: BrandMarkProps) {
  const { data } = useSiteConfig()
  const name = data?.system_name ?? 'Tavo'
  const logo = data?.logo || '/logo.svg'

  return (
    <span className={className}>
      <img src={logo} alt={name} className={iconClassName} />
      {showName && <span className={nameClassName}>{name}</span>}
    </span>
  )
}