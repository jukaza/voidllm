import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import apiClient from '../api/client'

export interface ModelRouteStep {
  id: string
  model_id: string
  position: number
  provider_id: string
  upstream_model: string
  is_enabled: boolean
  provider_name?: string
  provider_slug?: string
  provider_protocol?: string
  created_at?: string
}

export interface ModelRouteStepInput {
  provider_id: string
  upstream_model: string
  is_enabled: boolean
}

function modelRoutesKey(modelId: string) {
  return ['model-routes', modelId] as const
}

export function useModelRoutes(modelId: string) {
  return useQuery({
    queryKey: modelRoutesKey(modelId),
    queryFn: () =>
      apiClient<{ data: ModelRouteStep[] }>(`/models/${modelId}/routes`),
    enabled: Boolean(modelId),
  })
}

export function useReplaceModelRoutes() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      modelId,
      steps,
    }: {
      modelId: string
      steps: ModelRouteStepInput[]
    }) =>
      apiClient<{ data: ModelRouteStep[] }>(`/models/${modelId}/routes`, {
        method: 'PUT',
        body: JSON.stringify({ steps }),
      }),
    onSuccess: (_data, { modelId }) => {
      void queryClient.invalidateQueries({ queryKey: modelRoutesKey(modelId) })
      void queryClient.invalidateQueries({ queryKey: ['models'] })
    },
  })
}