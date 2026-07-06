import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import LoginPage from './pages/auth/LoginPage'
import AuthCallbackPage from './pages/auth/AuthCallbackPage'
import LandingPage from './pages/storefront/LandingPage'
import RegisterPage from './pages/storefront/RegisterPage'
import DashboardPage from './pages/DashboardPage'
import IntegrationsPage from './pages/IntegrationsPage'
import KeysPage from './pages/KeysPage'
import KeyUsagePage from './pages/KeyUsagePage'
import FinanceLayout from './pages/finance/FinanceLayout'
import FinanceOverviewPage from './pages/finance/FinanceOverviewPage'
import FinanceTopupsPage from './pages/finance/FinanceTopupsPage'
import FinanceLedgerPage from './pages/finance/FinanceLedgerPage'
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
import CostPage from './pages/analytics/CostPage'
import ChannelsPage from './pages/analytics/ChannelsPage'
import AnalyticsCostRedirect from './pages/analytics/AnalyticsRedirect'
import AccountSettingsPage from './pages/account/AccountSettingsPage'
import PlaygroundPage from './pages/PlaygroundPage'
import SystemUsersPage from './pages/SystemUsersPage'
import SubscriptionsPage from './pages/SubscriptionsPage'
import PlansPage from './pages/PlansPage'
import PlanPackagePage from './pages/PlanPackagePage'
import MySubscriptionsPage from './pages/MySubscriptionsPage'
import CatalogPage from './pages/CatalogPage'
import SystemSettingsPage from './pages/settings/SystemSettingsPage'
import LegalPage from './pages/storefront/LegalPage'
import { ToastProvider } from './hooks/useToast'
import { CatalogGate, PlaygroundGate } from './components/layout/FeatureGate'
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

export default function App() {
  return (
    <TranslationProvider>
      <QueryClientProvider client={queryClient}>
        <ToastProvider>
          <BrowserRouter>
            <Routes>
              <Route path="/" element={<LandingPage />} />
              <Route path="/login" element={<LoginPage />} />
              <Route path="/register" element={<RegisterPage />} />
              <Route path="/auth/callback" element={<AuthCallbackPage />} />
              <Route path="/legal/:kind" element={<LegalPage />} />
              <Route path="/key-usage" element={<KeyUsagePage />} />
              <Route path="/system-settings/*" element={<Navigate to="/settings" replace />} />
              <Route element={<RequireAuth />}>
                <Route path="/dashboard" element={<DashboardPage />} />
                <Route path="/integrations" element={<IntegrationsPage />} />
                <Route path="/dashboard/integrations" element={<Navigate to="/integrations" replace />} />
                <Route
                  path="/playground"
                  element={
                    <PlaygroundGate>
                      <PlaygroundPage />
                    </PlaygroundGate>
                  }
                />
                <Route
                  path="/catalog"
                  element={
                    <CatalogGate>
                      <CatalogPage />
                    </CatalogGate>
                  }
                />
                <Route path="/keys" element={<KeysPage />} />
                <Route path="/providers" element={<ProvidersPage />} />
                <Route path="/providers/new" element={<ProviderNewPage />} />
                <Route path="/providers/wizard" element={<ProviderWizardPage />} />
                <Route path="/providers/:id" element={<ProviderDetailPage />} />
                <Route path="/finance" element={<FinanceLayout />}>
                  <Route index element={<FinanceOverviewPage />} />
                  <Route path="topups" element={<FinanceTopupsPage />} />
                  <Route path="ledger" element={<FinanceLedgerPage />} />
                </Route>
                <Route path="/marketplace" element={<Navigate to="/finance/topups" replace />} />
                <Route path="/marketplace/*" element={<Navigate to="/finance/topups" replace />} />
                <Route path="/wallet" element={<WalletPage />} />
                <Route path="/plans" element={<PlansPage />} />
                <Route path="/plans/:packageId" element={<PlanPackagePage />} />
                <Route path="/my-subscriptions" element={<MySubscriptionsPage />} />
                <Route path="/models" element={<ModelsLayout />} />
                <Route path="/analytics" element={<AnalyticsLayout />}>
                  <Route index element={<AnalyticsOverviewPage />} />
                  <Route path="logs" element={<RequestLogsPage />} />
                  <Route path="channels" element={<ChannelsPage />} />
                  <Route path="profit" element={<ProfitPage />} />
                  <Route path="cost" element={<CostPage />} />
                </Route>
                <Route path="/usage" element={<Navigate to="/analytics" replace />} />
                <Route path="/usage/*" element={<Navigate to="/analytics" replace />} />
                <Route path="/cost-reports" element={<AnalyticsCostRedirect />} />
                <Route path="/account" element={<AccountSettingsPage />} />
                <Route path="/profile" element={<Navigate to="/account" replace />} />
                <Route path="/users" element={<SystemUsersPage />} />
                <Route path="/subscriptions" element={<SubscriptionsPage />} />
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