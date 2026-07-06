import { Navigate } from 'react-router-dom'
import { usePublicFeatures } from '../../hooks/useFeaturesSettings'

export function PlaygroundGate({ children }: { children: React.ReactNode }) {
  const { data, isLoading } = usePublicFeatures()
  if (isLoading) return null
  if (data?.modules.playground === false) {
    return <Navigate to="/dashboard" replace />
  }
  return <>{children}</>
}

export function CatalogGate({ children }: { children: React.ReactNode }) {
  const { data, isLoading } = usePublicFeatures()
  if (isLoading) return null
  if (data?.modules.public_catalog === false) {
    return <Navigate to="/dashboard" replace />
  }
  return <>{children}</>
}