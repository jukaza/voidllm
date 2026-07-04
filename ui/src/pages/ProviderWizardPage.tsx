import { useEffect } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'

/** Legacy route — redirects to Providers drawer (new-api style). */
export default function ProviderWizardPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const preset = searchParams.get('preset')

  useEffect(() => {
    if (preset) {
      navigate(`/providers/new?preset=${preset}`, { replace: true })
    } else {
      navigate('/providers/new', { replace: true })
    }
  }, [navigate, preset])

  return null
}