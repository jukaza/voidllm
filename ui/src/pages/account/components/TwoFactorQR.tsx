import { useEffect, useState } from 'react'
import QRCode from 'qrcode'

interface TwoFactorQRProps {
  otpauthUrl: string
}

export function TwoFactorQR({ otpauthUrl }: TwoFactorQRProps) {
  const [dataUrl, setDataUrl] = useState<string | null>(null)

  useEffect(() => {
    if (!otpauthUrl) {
      setDataUrl(null)
      return
    }
    let cancelled = false
    void QRCode.toDataURL(otpauthUrl, {
      width: 200,
      margin: 2,
      errorCorrectionLevel: 'M',
    })
      .then((url) => {
        if (!cancelled) setDataUrl(url)
      })
      .catch(() => {
        if (!cancelled) setDataUrl(null)
      })
    return () => {
      cancelled = true
    }
  }, [otpauthUrl])

  if (!dataUrl) return null

  return (
    <img
      src={dataUrl}
      alt=""
      width={200}
      height={200}
      className="mx-auto rounded-md bg-white p-2"
    />
  )
}