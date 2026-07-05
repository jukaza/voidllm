import { useEffect, useRef, useState } from 'react'
import { Dialog } from '../ui/Dialog'
import { Button } from '../ui/Button'
import { formatCost } from '../../lib/utils'
import { useTranslation } from '../../lib/i18n'
import apiClient from '../../api/client'
import type { SepayOrder } from '../../hooks/usePaymentSettings'

interface SePayPaymentDialogProps {
  open: boolean
  onClose: () => void
  order: SepayOrder | null
  onSuccess: () => void
}

export function SePayPaymentDialog({ open, onClose, order, onSuccess }: SePayPaymentDialogProps) {
  const { t } = useTranslation()
  const [copied, setCopied] = useState<string | null>(null)
  const [status, setStatus] = useState<'pending' | 'completed' | 'expired'>('pending')
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null)

  useEffect(() => {
    if (!open) {
      setStatus('pending')
      return
    }
    if (!order || status !== 'pending') return

    const poll = async () => {
      try {
        const res = await apiClient<{ status: string }>(`/me/topups/${order.trade_no}/status`)
        if (res.status === 'completed') {
          setStatus('completed')
          onSuccess()
          if (pollRef.current) clearInterval(pollRef.current)
        } else if (res.status === 'expired') {
          setStatus('expired')
          if (pollRef.current) clearInterval(pollRef.current)
        }
      } catch {
        // keep polling
      }
    }

    pollRef.current = setInterval(() => void poll(), 2500)
    void poll()
    return () => {
      if (pollRef.current) clearInterval(pollRef.current)
    }
  }, [open, order, status, onSuccess])

  function copyText(text: string, field: string) {
    void navigator.clipboard.writeText(text)
    setCopied(field)
    setTimeout(() => setCopied(null), 2000)
  }

  if (!order) return null

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={
        status === 'completed'
          ? t('wallet.sepay_success_title')
          : status === 'expired'
            ? t('wallet.sepay_expired_title')
            : t('wallet.sepay_title')
      }
      panelClassName="max-w-lg"
      footer={
        <Button variant={status === 'completed' ? 'primary' : 'secondary'} onClick={onClose}>
          {t('common.close')}
        </Button>
      }
    >
      {status === 'expired' ? (
        <div className="space-y-3 text-center py-4">
          <div className="text-4xl text-error">!</div>
          <p className="text-sm text-text-secondary">{t('wallet.sepay_expired_desc')}</p>
        </div>
      ) : status === 'completed' ? (
        <div className="space-y-3 text-center py-4">
          <div className="text-4xl text-success">✓</div>
          <p className="text-sm text-text-secondary">{t('wallet.sepay_success_desc')}</p>
          <div className="rounded-lg border border-border bg-bg-tertiary p-4 text-sm space-y-2 text-left">
            <div className="flex justify-between gap-4">
              <span className="text-text-tertiary">{t('wallet.sepay_trade_no')}</span>
              <code className="font-mono text-xs">{order.trade_no}</code>
            </div>
            <div className="flex justify-between gap-4">
              <span className="text-text-tertiary">{t('wallet.sepay_credit')}</span>
              <span className="text-success font-semibold">{formatCost(order.credit_amount)}</span>
            </div>
          </div>
        </div>
      ) : (
        <div className="space-y-4">
          <p className="text-sm text-text-tertiary">{t('wallet.sepay_instructions')}</p>
          <div className="flex flex-col sm:flex-row gap-4">
            <div className="flex justify-center p-3 border border-border rounded-lg bg-white shrink-0 mx-auto sm:mx-0">
              <img src={order.qr_url} alt="VietQR" className="w-48 h-48 object-contain" />
            </div>
            <div className="flex-1 space-y-2 text-sm">
              <CopyRow label={t('wallet.sepay_bank')} value={order.bank_name} raw={order.bank_name} field="bank" copied={copied} onCopy={copyText} copyLabel={t('common.copy')} />
              <CopyRow label={t('wallet.sepay_account')} value={order.account_number} raw={order.account_number} field="account" copied={copied} onCopy={copyText} copyLabel={t('common.copy')} mono />
              <CopyRow label={t('wallet.sepay_holder')} value={order.account_name} raw={order.account_name} field="name" copied={copied} onCopy={copyText} copyLabel={t('common.copy')} />
              <CopyRow label={t('wallet.sepay_pay_amount')} value={formatCost(order.pay_amount)} raw={String(order.pay_amount)} field="amount" copied={copied} onCopy={copyText} copyLabel={t('common.copy')} highlight />
              <CopyRow label={t('wallet.sepay_transfer_content')} value={order.trade_no} raw={order.trade_no} field="content" copied={copied} onCopy={copyText} copyLabel={t('common.copy')} mono warn />
              {order.bonus_amount > 0 && (
                <p className="text-xs text-success pt-1">
                  {t('wallet.sepay_bonus_preview', {
                    credit: formatCost(order.credit_amount),
                    bonus: formatCost(order.bonus_amount),
                  })}
                </p>
              )}
            </div>
          </div>
          <p className="text-xs text-text-tertiary animate-pulse">{t('wallet.sepay_waiting')}</p>
        </div>
      )}
    </Dialog>
  )
}

function CopyRow({
  label,
  value,
  raw,
  field,
  copied,
  onCopy,
  copyLabel,
  mono,
  highlight,
  warn,
}: {
  label: string
  value: string
  raw: string
  field: string
  copied: string | null
  onCopy: (text: string, field: string) => void
  copyLabel: string
  mono?: boolean
  highlight?: boolean
  warn?: boolean
}) {
  return (
    <div className={`flex items-center justify-between gap-2 rounded-md px-2 py-1.5 ${warn ? 'bg-warning/10' : 'bg-bg-tertiary'}`}>
      <div className="min-w-0">
        <div className="text-xs text-text-tertiary">{label}</div>
        <div className={`truncate ${mono ? 'font-mono text-xs' : ''} ${highlight ? 'text-success font-semibold' : 'font-medium'}`}>
          {value}
        </div>
      </div>
      <Button size="sm" variant="ghost" onClick={() => onCopy(raw, field)}>
        {copied === field ? '✓' : copyLabel}
      </Button>
    </div>
  )
}