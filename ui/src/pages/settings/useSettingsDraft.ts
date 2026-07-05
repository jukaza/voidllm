import { useCallback, useEffect, useState } from 'react'
import {
  DEFAULT_SETTINGS_DRAFT,
  SETTINGS_DRAFT_STORAGE_KEY,
  type SettingsDraft,
} from './settingsDraftTypes'

function readDraft(): SettingsDraft {
  try {
    const raw = localStorage.getItem(SETTINGS_DRAFT_STORAGE_KEY)
    if (!raw) return { ...DEFAULT_SETTINGS_DRAFT }
    const parsed = JSON.parse(raw) as Partial<SettingsDraft>
    return { ...DEFAULT_SETTINGS_DRAFT, ...parsed }
  } catch {
    return { ...DEFAULT_SETTINGS_DRAFT }
  }
}

export function useSettingsDraft() {
  const [draft, setDraftState] = useState<SettingsDraft>(readDraft)

  useEffect(() => {
    try {
      localStorage.setItem(SETTINGS_DRAFT_STORAGE_KEY, JSON.stringify(draft))
    } catch {
      // ignore quota errors
    }
  }, [draft])

  const setDraft = useCallback((patch: Partial<SettingsDraft>) => {
    setDraftState((prev) => ({ ...prev, ...patch }))
  }, [])

  const resetDraft = useCallback(() => {
    setDraftState({ ...DEFAULT_SETTINGS_DRAFT })
    try {
      localStorage.removeItem(SETTINGS_DRAFT_STORAGE_KEY)
    } catch {
      // ignore
    }
  }, [])

  return { draft, setDraft, resetDraft }
}