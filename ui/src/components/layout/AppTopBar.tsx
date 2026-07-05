import { SiteNoticeBell } from '../site/SiteNoticeBell'

/** Slim top bar above console content — notification bell on the right. */
export function AppTopBar() {
  return (
    <div className="sticky top-0 z-40 flex h-12 shrink-0 items-center justify-end border-b border-white/5 bg-bg-primary/90 px-8 backdrop-blur-sm">
      <SiteNoticeBell />
    </div>
  )
}