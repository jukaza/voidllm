import { useEffect, useState } from 'react'
import { LOCAL_STORAGE_KEY } from '../lib/constants'

export interface LiveSnapshot {
  rpm: number
  tpm: number
  active_count: number
  active_requests: Array<{
    model: string
    provider: string
    deployment?: string
    count: number
  }>
  recent_requests: Array<{
    timestamp: string
    model: string
    provider: string
    deployment_id?: string
    prompt_tokens: number
    completion_tokens: number
    status_code: number
  }>
}

export function useUsageLive() {
  const [live, setLive] = useState<LiveSnapshot | null>(null)
  const [connected, setConnected] = useState(false)

  useEffect(() => {
    const key = localStorage.getItem(LOCAL_STORAGE_KEY) ?? ''
    if (!key) return

    const controller = new AbortController()

    ;(async () => {
      try {
        const res = await fetch('/api/v1/usage/stream', {
          headers: {
            Authorization: `Bearer ${key}`,
            Accept: 'text/event-stream',
          },
          signal: controller.signal,
        })
        if (!res.ok || !res.body) {
          setConnected(false)
          return
        }
        setConnected(true)
        const reader = res.body.getReader()
        const decoder = new TextDecoder()
        let buffer = ''

        while (true) {
          const { done, value } = await reader.read()
          if (done) break
          buffer += decoder.decode(value, { stream: true })
          const chunks = buffer.split('\n\n')
          buffer = chunks.pop() ?? ''
          for (const chunk of chunks) {
            const dataLine = chunk.split('\n').find((line) => line.startsWith('data: '))
            if (!dataLine) continue
            try {
              setLive(JSON.parse(dataLine.slice(6)) as LiveSnapshot)
            } catch {
              // ignore malformed frames
            }
          }
        }
      } catch {
        if (!controller.signal.aborted) {
          setConnected(false)
        }
      }
    })()

    return () => {
      controller.abort()
      setConnected(false)
    }
  }, [])

  return { live, connected }
}