export type AnnouncementType = 'default' | 'ongoing' | 'success' | 'warning' | 'error'

export interface SiteAnnouncement {
  id: string
  content: string
  publish_date: string
  type: AnnouncementType
  extra?: string
}

/** Sample opening announcement with Zalo support link + QR (for preview / demo). */
export function createSampleOpeningAnnouncement(systemName = 'VoidLLM'): SiteAnnouncement {
  const zaloUrl = 'https://zalo.me/g/voidllm-hotro'
  const qrUrl = `https://api.qrserver.com/v1/create-qr-code/?size=240x240&data=${encodeURIComponent(zaloUrl)}`

  return createAnnouncement({
    type: 'success',
    extra: '*Hỗ trợ trong giờ hành chính: 8h–22h mỗi ngày.*',
    content: `## 🎉 Khai trường — Chào mừng đến với **${systemName}**

Chúng tôi chính thức mở cổng API marketplace. Ưu đãi dành cho khách hàng mới:

- **Tặng 50.000đ** khi đăng ký và nạp lần đầu
- Giảm *10%* phí API trong **7 ngày** đầu

---

**Cần hỗ trợ?** Tham gia nhóm Zalo:

👉 [Nhóm Zalo hỗ trợ khách hàng](${zaloUrl})

Quét mã QR bên dưới để vào nhóm nhanh:

![Quét Zalo để được hỗ trợ 24/7](${qrUrl})`,
  })
}

export function createAnnouncement(partial?: Partial<SiteAnnouncement>): SiteAnnouncement {
  return {
    id: crypto.randomUUID(),
    content: partial?.content ?? '',
    publish_date: partial?.publish_date ?? new Date().toISOString(),
    type: partial?.type ?? 'default',
    extra: partial?.extra ?? '',
  }
}

export function announcementsFingerprint(items: SiteAnnouncement[]): string {
  const normalized = [...items]
    .sort((a, b) => a.id.localeCompare(b.id))
    .map((item) => ({
      id: item.id,
      content: item.content.trim(),
      publish_date: item.publish_date,
      type: item.type,
      extra: (item.extra ?? '').trim(),
    }))
  return JSON.stringify(normalized)
}

const dotClasses: Record<AnnouncementType, string> = {
  default: 'bg-text-tertiary',
  ongoing: 'bg-info',
  success: 'bg-success',
  warning: 'bg-warning',
  error: 'bg-error',
}

export function announcementDotClass(type?: string): string {
  if (type && type in dotClasses) {
    return dotClasses[type as AnnouncementType]
  }
  return dotClasses.default
}

export function formatAnnouncementDate(iso: string, locale: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return ''
  return date.toLocaleString(locale === 'vi' ? 'vi-VN' : 'en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}