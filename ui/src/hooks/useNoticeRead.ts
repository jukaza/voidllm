import { useCallback, useMemo, useSyncExternalStore } from 'react'
import { LAST_READ_NOTICE_KEY } from '../lib/constants'
import { announcementsFingerprint, type SiteAnnouncement } from '../lib/announcements'

function getSnapshot(): string {
  return localStorage.getItem(LAST_READ_NOTICE_KEY) ?? ''
}

function subscribe(onStoreChange: () => void) {
  const handler = (e: StorageEvent) => {
    if (e.key === LAST_READ_NOTICE_KEY || e.key === null) onStoreChange()
  }
  window.addEventListener('storage', handler)
  window.addEventListener('tavo-notice-read', onStoreChange)
  return () => {
    window.removeEventListener('storage', handler)
    window.removeEventListener('tavo-notice-read', onStoreChange)
  }
}

export function useNoticeRead(announcements: SiteAnnouncement[]) {
  const lastRead = useSyncExternalStore(subscribe, getSnapshot, () => '')

  const fingerprint = useMemo(
    () => announcementsFingerprint(announcements),
    [announcements],
  )

  const isUnread = announcements.length > 0 && fingerprint !== lastRead
  const unreadCount = isUnread ? announcements.length : 0

  const markRead = useCallback(() => {
    if (announcements.length === 0) return
    localStorage.setItem(LAST_READ_NOTICE_KEY, fingerprint)
    window.dispatchEvent(new Event('tavo-notice-read'))
  }, [announcements.length, fingerprint])

  return { isUnread, unreadCount, markRead }
}