import { useEffect } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'

/** Legacy route — redirects to Providers drawer (new-api style). */
export default function ProviderWizardPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const preset = searchParams.get('preset')

  useEffect(() => {
    const q = new URLSearchParams({ add: '1' })
    if (preset) q.set('preset', preset)
    navigate(`/providers?${q.toString()}`, { replace: true })
  }, [navigate, preset])

  return null
}