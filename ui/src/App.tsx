import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import LoginPage from './pages/auth/LoginPage'
import LandingPage from './pages/storefront/LandingPage'
import RegisterPage from './pages/storefront/RegisterPage'
import DashboardPage from './pages/DashboardPage'
import KeysPage from './pages/KeysPage'
import MarketplacePage from './pages/MarketplacePage'
import ProvidersPage from './pages/ProvidersPage'
import ProviderDetailPage from './pages/ProviderDetailPage'
import ProviderNewPage from './pages/ProviderNewPage'
import ProviderWizardPage from './pages/ProviderWizardPage'
import WalletPage from './pages/WalletPage'
import ModelsLayout from './pages/ModelsLayout'
import AnalyticsLayout from './pages/analytics/AnalyticsLayout'
import AnalyticsOverviewPage from './pages/analytics/AnalyticsOverviewPage'
import RequestLogsPage from './pages/analytics/RequestLogsPage'
import ProfitPage from './pages/analytics/ProfitPage'
import ChannelsPage from './pages/analytics/ChannelsPage'
import ProfilePage from './pages/ProfilePage'
import PlaygroundPage from './pages/PlaygroundPage'
import SystemUsersPage from './pages/SystemUsersPage'
import CatalogPage from './pages/CatalogPage'
import SystemSettingsPage from './pages/settings/SystemSettingsPage'
import LegalPage from './pages/storefront/LegalPage'
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
      refetchOnWindowFocus: false,
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
              <Route path="/legal/:kind" element={<LegalPage />} />
              <Route path="/system-settings/*" element={<Navigate to="/settings" replace />} />
              <Route element={<RequireAuth />}>
                <Route path="/dashboard" element={<DashboardPage />} />
                <Route path="/playground" element={<PlaygroundPage />} />
                <Route path="/catalog" element={<CatalogPage />} />
                <Route path="/keys" element={<KeysPage />} />
                <Route path="/providers" element={<ProvidersPage />} />
                <Route path="/providers/new" element={<ProviderNewPage />} />
                <Route path="/providers/wizard" element={<ProviderWizardPage />} />
                <Route path="/providers/:id" element={<ProviderDetailPage />} />
                <Route path="/marketplace" element={<MarketplacePage />} />
                <Route path="/wallet" element={<WalletPage />} />
                <Route path="/models" element={<ModelsLayout />} />
                <Route path="/analytics" element={<AnalyticsLayout />}>
                  <Route index element={<AnalyticsOverviewPage />} />
                  <Route path="logs" element={<RequestLogsPage />} />
                  <Route path="channels" element={<ChannelsPage />} />
                  <Route path="profit" element={<ProfitPage />} />
                </Route>
                <Route path="/usage" element={<Navigate to="/analytics" replace />} />
                <Route path="/usage/*" element={<Navigate to="/analytics" replace />} />
                <Route path="/cost-reports" element={<Navigate to="/analytics/profit" replace />} />
                <Route path="/profile" element={<ProfilePage />} />
                <Route path="/users" element={<SystemUsersPage />} />
                <Route path="/settings" element={<SystemSettingsPage />} />
                <Route path="/settings/*" element={<Navigate to="/settings" replace />} />
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