import React from 'react'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ToastProvider } from '../hooks/useToast'
import { TranslationProvider } from '../lib/i18n'

vi.mock('../lib/lobe-icon', () => ({
  getLobeIcon: () => null,
}))

import ModelsPage from './ModelsPage'

interface MockModelResponse {
  id: string
  name: string
  type: string
  provider: string
  base_url: string
  max_context_tokens: number
  input_price_per_1m: number
  output_price_per_1m: number
  is_active: boolean
  is_public?: boolean
  source: string
  aliases: string[]
  created_at: string
  updated_at: string
  routing_strategy?: string
  bill_per_token?: boolean
  sell_input_per_1m?: number | null
  sell_output_per_1m?: number | null
}

function makeProduct(overrides: Partial<MockModelResponse> = {}): MockModelResponse {
  return {
    id: 'model-1',
    name: 'premium-chat',
    type: 'chat',
    provider: '',
    base_url: '',
    max_context_tokens: 0,
    input_price_per_1m: 0,
    output_price_per_1m: 0,
    is_active: true,
    is_public: false,
    source: 'api',
    aliases: [],
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    routing_strategy: 'fallback',
    bill_per_token: true,
    sell_input_per_1m: 3,
    sell_output_per_1m: 12,
    ...overrides,
  }
}

function makeWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })
  function Wrapper({ children }: { children: React.ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <TranslationProvider>
          <ToastProvider>{children}</ToastProvider>
        </TranslationProvider>
      </QueryClientProvider>
    )
  }
  return { queryClient, Wrapper }
}

function renderModelsPage() {
  localStorage.setItem('tavo_lang', 'en')
  const { Wrapper } = makeWrapper()
  return render(<ModelsPage />, { wrapper: Wrapper })
}

type FetchMockEntry = {
  matcher: (url: string) => boolean
  response: unknown
  method?: string
}

function setupFetchMock(entries: FetchMockEntry[], capturedBodies = new Map<string, string>()) {
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
      const method = init?.method ?? 'GET'
      if (init?.body && typeof init.body === 'string') {
        capturedBodies.set(`${method}:${url.replace(/^.*\/api\/v1/, '')}`, init.body)
      }
      for (const entry of entries) {
        if (entry.method && entry.method !== method) continue
        if (entry.matcher(url)) {
          return new Response(JSON.stringify(entry.response), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          })
        }
      }
      return new Response(JSON.stringify({ error: 'unmocked', url, method }), { status: 404 })
    }),
  )
}

function defaultEntries(modelsData?: { data: MockModelResponse[]; has_more: boolean }) {
  const models = modelsData ?? { data: [makeProduct()], has_more: false }
  return [
    {
      matcher: (u: string) => u.includes('/api/v1/models') && !u.includes('/routes') && !u.includes('/health'),
      response: models,
    },
    {
      matcher: (u: string) => u.includes('/routes'),
      response: { data: [{ id: 'step-1', model_id: 'model-1', position: 0, provider_id: 'p1', upstream_model: 'gpt-4', is_enabled: true }] },
    },
    {
      matcher: (u: string) => u.includes('/upstream-models'),
      response: { data: [{ provider_id: 'p1', provider_name: 'OpenAI', upstream_id: 'gpt-4', is_enabled: true }] },
    },
    {
      matcher: (u: string) => u.includes('/health'),
      response: { models: [] },
    },
  ] satisfies FetchMockEntry[]
}

describe('ModelsPage — product catalog', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('renders product stats and table', async () => {
    setupFetchMock(defaultEntries())
    renderModelsPage()

    expect(await screen.findByText('premium-chat')).toBeInTheDocument()
    expect(screen.getByText('Products')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /create product/i })).toBeInTheDocument()
  })

  it('opens create product dialog', async () => {
    setupFetchMock(defaultEntries())
    renderModelsPage()

    await userEvent.click(await screen.findByRole('button', { name: /create product/i }))
    const dialog = await screen.findByRole('dialog')
    expect(within(dialog).getByRole('heading', { name: /create product/i })).toBeInTheDocument()
    expect(within(dialog).getByLabelText(/product name/i)).toBeInTheDocument()
  })

  it('requires name and route steps on create', async () => {
    setupFetchMock(defaultEntries())
    renderModelsPage()

    await userEvent.click(await screen.findByRole('button', { name: /create product/i }))
    const dialog = await screen.findByRole('dialog')
    await userEvent.click(within(dialog).getByRole('button', { name: /^create product$/i }))

    await waitFor(() => {
      expect(within(dialog).getByText(/product name is required/i)).toBeInTheDocument()
      expect(within(dialog).getByText(/at least one combo route step/i)).toBeInTheDocument()
    })
  })

  it('opens edit dialog for API products', async () => {
    setupFetchMock(defaultEntries())
    renderModelsPage()

    const row = (await screen.findByText('premium-chat')).closest('tr')
    if (!row) throw new Error('row not found')
    await userEvent.click(within(row).getByTitle(/edit product/i))

    expect(await screen.findByRole('heading', { name: /edit product/i })).toBeInTheDocument()
  })
})