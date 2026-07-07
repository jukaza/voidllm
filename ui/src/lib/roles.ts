import type { MeResponse } from '../hooks/useMe'

export function isRootUser(me?: MeResponse | null): boolean {
  return me?.role === 'root'
}

export function isStaffUser(me?: MeResponse | null): boolean {
  return me?.role === 'admin' || me?.role === 'root' || me?.is_system_admin === true
}