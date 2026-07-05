import { useMemo } from 'react'
import { NavLink, Link } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { useMe } from '../../hooks/useMe'
import { useMyWallet } from '../../hooks/useWallet'
import { cn, formatCost } from '../../lib/utils'
import { LOCAL_STORAGE_KEY } from '../../lib/constants'
import { useTranslation } from '../../lib/i18n'
import { BrandMark } from '../brand/BrandMark'

function formatRole(role?: string): string {
  if (!role) return '...'
  return role.split('_').map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(' ')
}

interface NavItem {
  label: string
  path: string
  icon: React.ReactNode
  locked?: boolean
  minRole?: string
  end?: boolean
}

interface NavGroup {
  label: string
  items: NavItem[]
  minRole?: string
}

const roleLevel: Record<string, number> = {
  member: 0,
  system_admin: 1,
}

function hasMinRole(userRole: string, minRole?: string): boolean {
  if (!minRole) return true
  if (import.meta.env.DEV && !(userRole in roleLevel)) {
    console.warn(`[Sidebar] Unknown role "${userRole}" — defaulting to member visibility`)
  }
  return (roleLevel[userRole] ?? 0) >= (roleLevel[minRole] ?? 0)
}

const iconProps = {
  className: 'h-4 w-4 shrink-0',
  viewBox: '0 0 24 24',
  fill: 'none',
  stroke: 'currentColor',
  strokeWidth: 1.5,
  strokeLinecap: 'round' as const,
  strokeLinejoin: 'round' as const,
  'aria-hidden': true,
}

function IconDashboard() {
  return (
    <svg {...iconProps}>
      <rect x="3" y="3" width="7" height="7" rx="1" />
      <rect x="14" y="3" width="7" height="7" rx="1" />
      <rect x="3" y="14" width="7" height="7" rx="1" />
      <rect x="14" y="14" width="7" height="7" rx="1" />
    </svg>
  )
}

function IconTerminal() {
  return (
    <svg {...iconProps}>
      <polyline points="4 17 10 11 4 5" />
      <line x1="12" y1="19" x2="20" y2="19" />
    </svg>
  )
}

function IconKey() {
  return (
    <svg {...iconProps}>
      <path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4" />
    </svg>
  )
}


function IconCatalog() {
  return (
    <svg {...iconProps}>
      <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20" />
      <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z" />
      <line x1="8" y1="7" x2="16" y2="7" />
      <line x1="8" y1="11" x2="14" y2="11" />
    </svg>
  )
}

function IconCube() {
  return (
    <svg {...iconProps}>
      <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" />
      <polyline points="3.27 6.96 12 12.01 20.73 6.96" />
      <line x1="12" y1="22.08" x2="12" y2="12" />
    </svg>
  )
}

function IconBarChart() {
  return (
    <svg {...iconProps}>
      <line x1="18" y1="20" x2="18" y2="10" />
      <line x1="12" y1="20" x2="12" y2="4" />
      <line x1="6" y1="20" x2="6" y2="14" />
    </svg>
  )
}

function IconBuilding() {
  return (
    <svg {...iconProps}>
      <rect x="4" y="2" width="16" height="20" rx="2" ry="2" />
      <line x1="9" y1="22" x2="9" y2="2" />
      <line x1="15" y1="22" x2="15" y2="2" />
      <line x1="4" y1="12" x2="20" y2="12" />
    </svg>
  )
}

function IconPersonPlus() {
  return (
    <svg {...iconProps}>
      <path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
      <circle cx="8.5" cy="7" r="4" />
      <line x1="20" y1="8" x2="20" y2="14" />
      <line x1="23" y1="11" x2="17" y2="11" />
    </svg>
  )
}

function IconSettings() {
  return (
    <svg {...iconProps}>
      <circle cx="12" cy="12" r="3" />
      <path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42" />
    </svg>
  )
}

function IconWallet() {
  return (
    <svg {...iconProps}>
      <path d="M21 12V7H5a2 2 0 0 1 0-4h14v4" />
      <path d="M3 5v14a2 2 0 0 0 2 2h16v-5" />
      <path d="M18 12a2 2 0 0 0 0 4h4v-4Z" />
    </svg>
  )
}

function buildNavigation(t: any): NavGroup[] {
  return [
    {
      label: t('sidebar.overview'),
      items: [
        { label: t('sidebar.dashboard'), path: '/dashboard', icon: <IconDashboard /> },
        { label: t('sidebar.playground'), path: '/playground', icon: <IconTerminal /> },
        { label: t('sidebar.wallet'), path: '/wallet', icon: <IconWallet /> },
      ],
    },
    {
      label: t('sidebar.manage'),
      items: [
        { label: t('sidebar.catalog'), path: '/catalog', icon: <IconCatalog /> },
        { label: t('sidebar.keys'), path: '/keys', icon: <IconKey /> },
        { label: t('sidebar.analytics'), path: '/analytics', icon: <IconBarChart /> },
      ],
    },
    {
      label: t('sidebar.system'),
      minRole: 'system_admin',
      items: [
        { label: t('sidebar.users'), path: '/users', icon: <IconPersonPlus /> },
        { label: t('sidebar.providers'), path: '/providers', icon: <IconBuilding /> },
        { label: t('sidebar.models'), path: '/models', icon: <IconCube /> },
        { label: t('sidebar.topups'), path: '/marketplace', icon: <IconWallet /> },
        { label: t('sidebar.settings'), path: '/settings', icon: <IconSettings /> },
      ],
    },
  ]
}

function LockIcon() {
  return (
    <svg
      aria-hidden="true"
      className="h-3 w-3 shrink-0 opacity-50"
      fill="none"
      viewBox="0 0 24 24"
      stroke="currentColor"
      strokeWidth={2.5}
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
      />
    </svg>
  )
}

export function Sidebar() {
  const { data } = useMe()
  const { data: wallet } = useMyWallet()
  const queryClient = useQueryClient()
  const { language, setLanguage, t } = useTranslation()

  const userRole = data?.role ?? 'member'

  const visibleGroups = useMemo(() => {
    const navigation = buildNavigation(t)
    return navigation
      .filter(group => hasMinRole(userRole, group.minRole))
      .map(group => ({
        ...group,
        items: group.items.filter(item => hasMinRole(userRole, item.minRole)),
      }))
      .filter(group => group.items.length > 0)
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [userRole, t])

  return (
    <aside
      aria-label="Main navigation"
      className="w-[13rem] bg-bg-secondary border-r border-white/5 flex flex-col fixed h-screen z-50"
    >
      {/* Logo */}
      <div className="px-3 py-2.5 border-b border-white/5 shrink-0">
        <a href="/" className="no-underline">
          <BrandMark
            nameClassName="gradient-text text-base font-bold truncate"
            iconClassName="h-6 w-6 shrink-0"
          />
        </a>
      </div>

      {/* Navigation */}
      <nav className="void-scroll flex-1 flex flex-col gap-px px-2 py-2 overflow-y-auto">
        {visibleGroups.map((group, groupIndex) => (
            <div key={group.label || `group-${groupIndex}`}>
              {groupIndex > 0 && (
                <div className="h-px bg-white/10 my-1.5 mx-1" />
              )}
              {group.label && (
                <div
                  className={cn(
                    'px-2 py-1 text-[10px] font-medium uppercase tracking-wider text-text-tertiary/60',
                    groupIndex > 0 && 'mt-0.5',
                  )}
                >
                  {group.label}
                </div>
              )}
              {group.items.map((item) =>
                item.locked ? (
                  <NavLink
                    key={item.path}
                    to={item.path}
                    className="flex items-center gap-2 px-2 py-1.5 rounded-md text-[13px] opacity-50 hover:opacity-70 transition-opacity"
                  >
                    {item.icon}
                    <span className="flex-1 truncate">{item.label}</span>
                    <LockIcon />
                  </NavLink>
                ) : (
                  <NavLink
                    key={item.path}
                    to={item.path}
                    end={item.end !== undefined ? item.end : item.path === '/'}
                    className={({ isActive }) =>
                      cn(
                        'flex items-center gap-2 px-2 py-1.5 rounded-md text-[13px] no-underline transition-colors duration-150',
                        isActive
                          ? 'bg-accent/15 text-accent font-medium'
                          : 'text-text-secondary hover:bg-bg-tertiary hover:text-text-primary',
                      )
                    }
                  >
                    {item.icon}
                    <span className="flex-1 truncate">{item.label}</span>
                  </NavLink>
                )
              )}
            </div>
          ))}
      </nav>

      {/* Footer */}
      <div className="shrink-0 border-t border-white/5 px-2 py-2 space-y-1.5">
        <Link
          to="/wallet"
          className="flex items-center justify-between gap-2 rounded-md border border-white/5 bg-white/[0.02] px-2 py-1.5 no-underline transition-colors hover:border-accent/30 hover:bg-accent/5"
        >
          <span className="text-[10px] text-text-tertiary truncate">{t('wallet.balance')}</span>
          <span className="text-xs font-semibold text-text-primary tabular-nums shrink-0">
            {wallet != null ? formatCost(wallet.balance) : '—'}
          </span>
        </Link>

        <div className="flex items-center gap-1.5 min-w-0">
          <Link
            to="/profile"
            className="min-w-0 flex-1 truncate text-[11px] text-text-secondary hover:text-text-primary transition-colors no-underline"
            title={data?.display_name || data?.email || '...'}
          >
            {data?.display_name || data?.email || '...'}
          </Link>
          <span className="shrink-0 rounded bg-accent/15 px-1 py-px text-[9px] font-semibold text-accent uppercase">
            {formatRole(data?.role)}
          </span>
        </div>

        <div className="flex items-center gap-1">
          <div className="flex gap-px rounded border border-white/5 bg-white/5 p-px">
            <button
              type="button"
              onClick={() => setLanguage('vi')}
              className={cn(
                'px-1.5 py-px text-[9px] font-bold rounded-sm cursor-pointer transition-colors',
                language === 'vi' ? 'bg-accent text-white' : 'text-text-tertiary hover:text-text-primary',
              )}
            >
              VI
            </button>
            <button
              type="button"
              onClick={() => setLanguage('en')}
              className={cn(
                'px-1.5 py-px text-[9px] font-bold rounded-sm cursor-pointer transition-colors',
                language === 'en' ? 'bg-accent text-white' : 'text-text-tertiary hover:text-text-primary',
              )}
            >
              EN
            </button>
          </div>
          <Link
            to="/profile"
            className="flex-1 py-1 rounded-md border border-white/10 text-[10px] text-text-secondary text-center no-underline transition-colors hover:border-accent/40 hover:text-text-primary"
          >
            {t('sidebar.profile')}
          </Link>
          <button
            type="button"
            onClick={() => {
              localStorage.removeItem(LOCAL_STORAGE_KEY)
              queryClient.clear()
              window.location.href = '/login'
            }}
            className="flex-1 py-1 rounded-md border border-white/10 text-[10px] text-text-secondary cursor-pointer transition-colors hover:border-error hover:text-error"
          >
            {t('sidebar.logout')}
          </button>
        </div>
      </div>
    </aside>
  )
}
