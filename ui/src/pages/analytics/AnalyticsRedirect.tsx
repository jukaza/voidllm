import { Navigate } from 'react-router-dom'
import { useMe } from '../../hooks/useMe'

/** Sends legacy /cost-reports links to the role-appropriate analytics tab. */
export default function AnalyticsCostRedirect() {
  const { data: me } = useMe()
  const isAdmin = me?.is_system_admin ?? false
  return <Navigate to={isAdmin ? '/analytics/profit' : '/analytics/cost'} replace />
}