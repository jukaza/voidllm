import { useCallback, useEffect, useState } from 'react'
import {
  ACCOUNT_DRAFT_STORAGE_KEY,
  DEFAULT_ACCOUNT_DRAFT,
  type AccountDraft,
} from './accountDraftTypes'

function readDraft(): AccountDraft {
  try {
    const raw = localStorage.getItem(ACCOUNT_DRAFT_STORAGE_KEY)
    if (!raw) return { ...DEFAULT_ACCOUNT_DRAFT }
    return { ...DEFAULT_ACCOUNT_DRAFT, ...JSON.parse(raw) }
  } catch {
    return { ...DEFAULT_ACCOUNT_DRAFT }
  }
}

export function useAccountDraft() {
  const [draft, setDraftState] = useState<AccountDraft>(readDraft)

  useEffect(() => {
    try {
      localStorage.setItem(ACCOUNT_DRAFT_STORAGE_KEY, JSON.stringify(draft))
    } catch {
      // ignore
    }
  }, [draft])

  const setDraft = useCallback((patch: Partial<AccountDraft>) => {
    setDraftState((prev) => ({ ...prev, ...patch }))
  }, [])

  const resetDraft = useCallback(() => {
    setDraftState({ ...DEFAULT_ACCOUNT_DRAFT })
    try {
      localStorage.removeItem(ACCOUNT_DRAFT_STORAGE_KEY)
    } catch {
      // ignore
    }
  }, [])

  return { draft, setDraft, resetDraft }
}