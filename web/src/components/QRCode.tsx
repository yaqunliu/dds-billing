import { useState, useEffect, useRef, useCallback } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { QRCodeSVG } from 'qrcode.react'
import { getOrder } from '../api'
import { QRCODE_MESSAGES, pickLocale } from '../utils/locale'

interface Props {
  orderNo: string
  qrCodeUrl: string
  payUrl: string
  expiresAt: string
  paymentType: 'wxpay' | 'alipay'
  isDark: boolean
  lang: 'zh' | 'en'
  onClose: () => void
}

export default function QRCodeModal({
  orderNo,
  qrCodeUrl,
  payUrl,
  expiresAt,
  paymentType,
  isDark,
  lang,
  onClose,
}: Props) {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const [remaining, setRemaining] = useState(0)
  const [status, setStatus] = useState<'pending' | 'paid' | 'completed' | 'failed' | 'expired'>('pending')
  const pollRef = useRef<ReturnType<typeof setInterval>>(undefined)
  const timerRef = useRef<ReturnType<typeof setInterval>>(undefined)

  // Countdown
  useEffect(() => {
    const updateRemaining = () => {
      const diff = Math.max(0, Math.floor((new Date(expiresAt).getTime() - Date.now()) / 1000))
      setRemaining(diff)
      if (diff <= 0) setStatus('expired')
    }
    updateRemaining()
    timerRef.current = setInterval(updateRemaining, 1000)
    return () => clearInterval(timerRef.current)
  }, [expiresAt])

  // Poll order status
  const poll = useCallback(async () => {
    try {
      const res = await getOrder(orderNo)
      if (res.data.code === 0) {
        const s = res.data.data.status
        if (s === 'paid' || s === 'recharging' || s === 'completed') {
          setStatus(s === 'paid' || s === 'recharging' ? 'paid' : 'completed')
        } else if (s === 'failed') {
          setStatus('failed')
        }
      }
    } catch { /* ignore */ }
  }, [orderNo])

  useEffect(() => {
    if (status === 'pending') {
      pollRef.current = setInterval(poll, 2000)
    }
    return () => clearInterval(pollRef.current)
  }, [status, poll])

  // Redirect on completed
  useEffect(() => {
    if (status === 'completed' || status === 'paid') {
      const timer = setTimeout(() => {
        const params = searchParams.toString()
        navigate(`/pay/result?order_no=${orderNo}&status=${status}${params ? '&' + params : ''}`)
      }, status === 'completed' ? 1000 : 3000)
      return () => clearTimeout(timer)
    }
  }, [status, navigate, orderNo, searchParams])

  const minutes = Math.floor(remaining / 60)
  const seconds = remaining % 60
  const t = pickLocale(QRCODE_MESSAGES, lang)
  const payLabel = t.paymentLabels[paymentType]
  const payColor = paymentType === 'wxpay' ? 'text-green-500' : 'text-blue-500'

  // 优先使用渠道返回的二维码图片，没有则从 payUrl 生成
  const hasImageQR = !!qrCodeUrl
  const qrValue = qrCodeUrl || payUrl

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm" onClick={onClose}>
      <div
        className={`relative w-[340px] rounded-2xl shadow-2xl p-6 ${isDark ? 'bg-gray-800' : 'bg-white'}`}
        onClick={e => e.stopPropagation()}
      >
        {/* Close button */}
        <button
          onClick={onClose}
          className={`absolute top-3 right-3 w-8 h-8 rounded-full flex items-center justify-center transition-colors
            ${isDark ? 'hover:bg-gray-700 text-gray-400' : 'hover:bg-gray-100 text-gray-500'}`}
        >
          ✕
        </button>

        {/* Title */}
        <h3 className={`text-center text-lg font-semibold mb-1 ${payColor}`}>
          {t.scanTitle(payLabel)}
        </h3>

        {/* Status display */}
        {status === 'pending' && (
          <>
            {/* QR Code */}
            <div className={`mx-auto my-4 w-[220px] h-[220px] rounded-xl overflow-hidden border flex items-center justify-center
              ${isDark ? 'border-gray-700 bg-white' : 'border-gray-200 bg-white'}`}
            >
              {hasImageQR ? (
                <img src={qrCodeUrl} alt={t.qrAlt} className="w-full h-full object-contain" />
              ) : qrValue ? (
                <QRCodeSVG value={qrValue} size={200} level="M" />
              ) : (
                <p className="text-gray-400 text-sm">{t.qrLoading}</p>
              )}
            </div>

            {/* Countdown */}
            <div className="text-center">
              <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                {t.countdownText(`${minutes}:${seconds.toString().padStart(2, '0')}`)}
              </p>
              <p className={`text-xs mt-2 ${isDark ? 'text-gray-500' : 'text-gray-400'}`}>
                {t.orderNo}: {orderNo}
              </p>
            </div>
          </>
        )}

        {status === 'paid' && (
          <div className="text-center py-10">
            <div className="text-5xl mb-4">&#10003;</div>
            <p className="text-lg font-medium text-green-500">{t.paymentSuccess}</p>
            <p className={`text-sm mt-2 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{t.recharging}</p>
          </div>
        )}

        {status === 'completed' && (
          <div className="text-center py-10">
            <div className="text-5xl mb-4">&#10003;</div>
            <p className="text-lg font-medium text-green-500">{t.rechargeComplete}</p>
            <p className={`text-sm mt-2 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{t.redirecting}</p>
          </div>
        )}

        {status === 'expired' && (
          <div className="text-center py-10">
            <div className="text-5xl mb-4 text-gray-400">&#9201;</div>
            <p className={`text-lg font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>{t.orderExpired}</p>
            <button
              onClick={onClose}
              className="mt-4 px-6 py-2 rounded-lg bg-blue-600 text-white text-sm hover:bg-blue-700"
            >
              {t.placeOrderAgain}
            </button>
          </div>
        )}

        {status === 'failed' && (
          <div className="text-center py-10">
            <div className="text-5xl mb-4 text-red-400">&#10007;</div>
            <p className="text-lg font-medium text-red-500">{t.rechargeFailed}</p>
            <p className={`text-sm mt-2 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
              {t.supportTip}
            </p>
            <button
              onClick={onClose}
              className="mt-4 px-6 py-2 rounded-lg bg-blue-600 text-white text-sm hover:bg-blue-700"
            >
              {t.back}
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
