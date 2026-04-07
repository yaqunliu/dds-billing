import { useState, useEffect } from 'react'
import { useUrlParams } from '../hooks/useUrlParams'
import { getConfig, createOrder, type AppConfig } from '../api'
import QRCodeModal from '../components/QRCode'

const QUICK_AMOUNTS = [10, 20, 50, 100, 200, 500]

export default function Pay() {
  const { token, theme } = useUrlParams()
  const [config, setConfig] = useState<AppConfig | null>(null)
  const [selectedAmount, setSelectedAmount] = useState<number | null>(null)
  const [customAmount, setCustomAmount] = useState('')
  const [paymentType, setPaymentType] = useState<'wxpay' | 'alipay'>('wxpay')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  // QR code modal state
  const [orderNo, setOrderNo] = useState('')
  const [qrCodeUrl, setQrCodeUrl] = useState('')
  const [expiresAt, setExpiresAt] = useState('')
  const [showQR, setShowQR] = useState(false)

  useEffect(() => {
    getConfig().then(res => {
      if (res.data.code === 0) setConfig(res.data.data)
    }).catch(() => {})
  }, [])

  const amount = customAmount ? parseFloat(customAmount) : selectedAmount
  const isAmountValid = amount !== null && !isNaN(amount!) &&
    amount! >= (config?.min_amount ?? 1) &&
    amount! <= (config?.max_amount ?? 20000)

  const enabledTypes = config?.enabled_types ?? ['wxpay', 'alipay']

  const handleSelectAmount = (val: number) => {
    setSelectedAmount(val)
    setCustomAmount('')
    setError('')
  }

  const handleCustomAmountChange = (val: string) => {
    setCustomAmount(val)
    setSelectedAmount(null)
    setError('')
  }

  const handleSubmit = async () => {
    if (!amount || !isAmountValid) {
      setError(`请输入 ${config?.min_amount ?? 1} - ${config?.max_amount ?? 20000} 之间的金额`)
      return
    }
    if (!token) {
      setError('缺少用户凭证，请从平台入口进入')
      return
    }

    setLoading(true)
    setError('')
    try {
      const res = await createOrder({ token, amount, payment_type: paymentType })
      if (res.data.code === 0) {
        setOrderNo(res.data.data.order_no)
        setQrCodeUrl(res.data.data.qr_code_url)
        setExpiresAt(res.data.data.expires_at)
        setShowQR(true)
      } else {
        setError(res.data.message || '创建订单失败')
      }
    } catch (err: any) {
      setError(err.response?.data?.message || '网络错误，请重试')
    } finally {
      setLoading(false)
    }
  }

  const isDark = theme === 'dark'

  return (
    <div className={`min-h-screen transition-colors ${isDark ? 'bg-gray-900 text-gray-100' : 'bg-gray-50 text-gray-900'}`}>
      <div className="max-w-md mx-auto px-4 py-8">
        <h1 className="text-2xl font-bold text-center mb-8">账户充值</h1>

        {/* Quick amount selection */}
        <div className="mb-6">
          <label className={`block text-sm font-medium mb-3 ${isDark ? 'text-gray-300' : 'text-gray-700'}`}>
            选择金额
          </label>
          <div className="grid grid-cols-3 gap-3">
            {QUICK_AMOUNTS.map(val => (
              <button
                key={val}
                onClick={() => handleSelectAmount(val)}
                className={`py-3 rounded-lg text-lg font-medium transition-all border
                  ${selectedAmount === val && !customAmount
                    ? 'bg-blue-600 text-white border-blue-600 shadow-md'
                    : isDark
                      ? 'bg-gray-800 text-gray-200 border-gray-700 hover:border-blue-500'
                      : 'bg-white text-gray-700 border-gray-200 hover:border-blue-400'
                  }`}
              >
                ¥{val}
              </button>
            ))}
          </div>
        </div>

        {/* Custom amount */}
        <div className="mb-6">
          <label className={`block text-sm font-medium mb-2 ${isDark ? 'text-gray-300' : 'text-gray-700'}`}>
            自定义金额
          </label>
          <div className="relative">
            <span className={`absolute left-3 top-1/2 -translate-y-1/2 text-lg ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>¥</span>
            <input
              type="number"
              value={customAmount}
              onChange={e => handleCustomAmountChange(e.target.value)}
              placeholder={`${config?.min_amount ?? 1} - ${config?.max_amount ?? 20000}`}
              className={`w-full pl-8 pr-4 py-3 rounded-lg border text-lg transition-colors
                ${isDark
                  ? 'bg-gray-800 border-gray-700 text-gray-100 placeholder-gray-500 focus:border-blue-500'
                  : 'bg-white border-gray-200 text-gray-900 placeholder-gray-400 focus:border-blue-500'
                } focus:outline-none focus:ring-2 focus:ring-blue-500/20`}
            />
          </div>
        </div>

        {/* Payment type */}
        <div className="mb-8">
          <label className={`block text-sm font-medium mb-3 ${isDark ? 'text-gray-300' : 'text-gray-700'}`}>
            支付方式
          </label>
          <div className="flex gap-3">
            {enabledTypes.includes('wxpay') && (
              <button
                onClick={() => setPaymentType('wxpay')}
                className={`flex-1 py-3 rounded-lg font-medium transition-all border flex items-center justify-center gap-2
                  ${paymentType === 'wxpay'
                    ? 'bg-green-600 text-white border-green-600 shadow-md'
                    : isDark
                      ? 'bg-gray-800 text-gray-200 border-gray-700 hover:border-green-500'
                      : 'bg-white text-gray-700 border-gray-200 hover:border-green-400'
                  }`}
              >
                <WechatIcon />
                微信支付
              </button>
            )}
            {enabledTypes.includes('alipay') && (
              <button
                onClick={() => setPaymentType('alipay')}
                className={`flex-1 py-3 rounded-lg font-medium transition-all border flex items-center justify-center gap-2
                  ${paymentType === 'alipay'
                    ? 'bg-blue-500 text-white border-blue-500 shadow-md'
                    : isDark
                      ? 'bg-gray-800 text-gray-200 border-gray-700 hover:border-blue-400'
                      : 'bg-white text-gray-700 border-gray-200 hover:border-blue-400'
                  }`}
              >
                <AlipayIcon />
                支付宝
              </button>
            )}
          </div>
        </div>

        {/* Error */}
        {error && (
          <div className="mb-4 p-3 rounded-lg bg-red-500/10 border border-red-500/20 text-red-500 text-sm">
            {error}
          </div>
        )}

        {/* Submit */}
        <button
          onClick={handleSubmit}
          disabled={loading || !isAmountValid}
          className={`w-full py-4 rounded-lg text-lg font-bold transition-all
            ${loading || !isAmountValid
              ? 'bg-gray-400 text-gray-200 cursor-not-allowed'
              : 'bg-blue-600 text-white hover:bg-blue-700 active:bg-blue-800 shadow-lg'
            }`}
        >
          {loading ? '创建订单中...' : amount ? `支付 ¥${amount.toFixed(2)}` : '请选择金额'}
        </button>
      </div>

      {/* QR Code Modal */}
      {showQR && (
        <QRCodeModal
          orderNo={orderNo}
          qrCodeUrl={qrCodeUrl}
          expiresAt={expiresAt}
          paymentType={paymentType}
          isDark={isDark}
          onClose={() => setShowQR(false)}
        />
      )}
    </div>
  )
}

function WechatIcon() {
  return (
    <svg viewBox="0 0 24 24" className="w-5 h-5 fill-current">
      <path d="M8.691 2.188C3.891 2.188 0 5.476 0 9.53c0 2.212 1.17 4.203 3.002 5.55a.59.59 0 0 1 .213.665l-.39 1.48c-.019.07-.048.141-.048.213 0 .163.13.295.29.295a.326.326 0 0 0 .167-.054l1.903-1.114a.864.864 0 0 1 .717-.098 10.16 10.16 0 0 0 2.837.403c.276 0 .543-.027.811-.05-.857-2.578.157-4.972 1.932-6.446 1.703-1.415 3.882-1.98 5.853-1.838-.576-3.583-4.196-6.348-8.596-6.348zM5.785 5.991c.642 0 1.162.529 1.162 1.18a1.17 1.17 0 0 1-1.162 1.178A1.17 1.17 0 0 1 4.623 7.17c0-.651.52-1.18 1.162-1.18zm5.813 0c.642 0 1.162.529 1.162 1.18a1.17 1.17 0 0 1-1.162 1.178 1.17 1.17 0 0 1-1.162-1.178c0-.651.52-1.18 1.162-1.18zm3.868 2.8c-3.986 0-7.229 2.694-7.229 6.017 0 3.323 3.243 6.017 7.229 6.017.85 0 1.67-.136 2.434-.38a.72.72 0 0 1 .59.083l1.563.917a.27.27 0 0 0 .138.045c.133 0 .24-.11.24-.245 0-.059-.024-.118-.04-.176l-.32-1.218a.49.49 0 0 1 .176-.548C22.093 17.786 23 16.19 23 14.808c0-3.323-3.243-6.017-7.534-6.017zm-2.293 3.399c.528 0 .957.435.957.971a.964.964 0 0 1-.957.971.964.964 0 0 1-.957-.97c0-.537.43-.972.957-.972zm4.586 0c.528 0 .957.435.957.971a.964.964 0 0 1-.957.971.964.964 0 0 1-.957-.97c0-.537.43-.972.957-.972z" />
    </svg>
  )
}

function AlipayIcon() {
  return (
    <svg viewBox="0 0 24 24" className="w-5 h-5 fill-current">
      <path d="M21.422 14.762c-1.758-.39-3.29-.86-3.29-.86s.723-1.744 1.003-3.533h-3.55v-1.27h4.26V8.14h-4.26V5.75H13.6v2.39H9.54v.96h4.06v1.27H9.93v.96h6.54a12.3 12.3 0 0 1-.62 1.975s-2.771-.916-4.384-.674c-1.613.243-2.873 1.216-2.873 2.433 0 1.865 1.86 2.992 4.196 2.992 1.73 0 3.122-.678 4.133-1.59 1.476 1.043 3.456 2.103 5.078 2.724V3.6A3.6 3.6 0 0 0 18.4 0H5.6A3.6 3.6 0 0 0 2 3.6v16.8A3.6 3.6 0 0 0 5.6 24h12.8a3.6 3.6 0 0 0 3.6-3.6v-5.138c-.19-.16-.39-.32-.578-.5zm-8.343 3.353c-2.393 0-3.058-1.2-3.058-2.1 0-.9.756-1.76 2.194-1.76 1.725 0 3.277.84 3.277.84a6.7 6.7 0 0 1-2.413 3.02z" />
    </svg>
  )
}
