export const DEFAULT_SITE_LOGO = '/logo.svg'
export const UPLOADED_LOGO_PREFIX = '/uploads/site/'

export function isUploadedSiteLogo(path?: string | null): boolean {
  return Boolean(path?.startsWith(UPLOADED_LOGO_PREFIX))
}

export function logoSrc(path: string, cacheBust?: number): string {
  if (!cacheBust) return path
  const joiner = path.includes('?') ? '&' : '?'
  return `${path}${joiner}v=${cacheBust}`
}