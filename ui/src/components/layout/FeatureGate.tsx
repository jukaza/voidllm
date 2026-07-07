import { Navigate } from 'react-router-dom'
import { usePublicFeatures } from '../../hooks/useFeaturesSettings'

function GateLoading() {
  return (
    <div className="flex min-h-[40vh] items-center justify-center text-sm text-text-tertiary">
      Loading…
    </div>
  )
}

export function PlaygroundGate({ children }: { children: React.ReactNode }) {
  const { data, isLoading } = usePublicFeatures()
  if (isLoading) return <GateLoading />
  if (data?.modules.playground === false) {
    return <Navigate to="/dashboard" replace />
  }
  return <>{children}</>
}

export function CatalogGate({ children }: { children: React.ReactNode }) {
  const { data, isLoading } = usePublicFeatures()
  if (isLoading) return <GateLoading />
  if (data?.modules.public_catalog === false) {
    return <Navigate to="/dashboard" replace />
  }
  return <>{children}</>
}