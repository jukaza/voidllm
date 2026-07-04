import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import LoginPage from './pages/auth/LoginPage'
import LandingPage from './pages/storefront/LandingPage'
import RegisterPage from './pages/storefront/RegisterPage'
import DashboardPage from './pages/DashboardPage'
import KeysPage from './pages/KeysPage'
import MarketplacePage from './pages/MarketplacePage'
import ProvidersPage from './pages/ProvidersPage'
import ProviderWizardPage from './pages/ProviderWizardPage'
import WalletPage from './pages/WalletPage'
import ModelsLayout from './pages/ModelsLayout'
import UsageLayout from './pages/usage/UsageLayout'
import UsageOverviewPage from './pages/usage/UsageOverviewPage'
import LLMUsagePage from './pages/usage/LLMUsagePage'
import CostReportsPage from './pages/CostReportsPage'
import ProfilePage from './pages/ProfilePage'
import PlaygroundPage from './pages/PlaygroundPage'
import SystemUsersPage from './pages/SystemUsersPage'
import CatalogPage from './pages/CatalogPage'
import { ToastProvider } from './hooks/useToast'
import { Shell } from './components/layout/Shell'
import { PageHeader } from './components/ui/PageHeader'
import { LOCAL_STORAGE_KEY } from './lib/constants'
import { TranslationProvider } from './lib/i18n.tsx'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      staleTime: 30_000,
    },
  },
})

function PlaceholderPage({ title, description }: { title: string; description?: string }) {
  return (
    <>
      <PageHeader title={title} description={description} />
      <div className="rounded-lg border border-border bg-bg-secondary p-12 text-center">
        <p className="text-sm text-text-tertiary">Coming soon</p>
      </div>
    </>
  )
}

function RequireAuth() {
  const token = localStorage.getItem(LOCAL_STORAGE_KEY)
  if (!token) return <Navigate to="/" replace />
  return <Shell />
}

function HomeRoute() {
  const token = localStorage.getItem(LOCAL_STORAGE_KEY)
  if (!token) return <LandingPage />
  return <Navigate to="/dashboard" replace />
}

export default function App() {
  return (
    <TranslationProvider>
      <QueryClientProvider client={queryClient}>
        <ToastProvider>
          <BrowserRouter>
            <Routes>
              <Route path="/" element={<HomeRoute />} />
              <Route path="/login" element={<LoginPage />} />
              <Route path="/register" element={<RegisterPage />} />
              <Route element={<RequireAuth />}>
                <Route path="dashboard" element={<DashboardPage />} />
                <Route path="playground" element={<PlaygroundPage />} />
                <Route path="catalog" element={<CatalogPage />} />
                <Route path="keys" element={<KeysPage />} />
                <Route path="providers" element={<ProvidersPage />} />
                <Route path="providers/wizard" element={<ProviderWizardPage />} />
                <Route path="marketplace" element={<MarketplacePage />} />
                <Route path="wallet" element={<WalletPage />} />
                <Route path="models" element={<ModelsLayout />} />
                <Route path="usage" element={<UsageLayout />}>
                  <Route index element={<UsageOverviewPage />} />
                  <Route path="llm" element={<LLMUsagePage />} />
                </Route>
                <Route path="cost-reports" element={<CostReportsPage />} />
                <Route path="profile" element={<ProfilePage />} />
                <Route path="users" element={<SystemUsersPage />} />
                <Route
                  path="*"
                  element={
                    <PlaceholderPage
                      title="Not Found"
                      description="This page does not exist."
                    />
                  }
                />
              </Route>
            </Routes>
          </BrowserRouter>
        </ToastProvider>
      </QueryClientProvider>
    </TranslationProvider>
  )
}