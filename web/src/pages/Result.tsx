import { useEffect, useState } from 'react'
import { useSearchParams, useNavigate } from 'react-router-dom'
import { getOrder, type OrderData } from '../api'
import { useUrlParams } from '../hooks/useUrlParams'

export default function Result() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const { theme } = useUrlParams()
  const isDark = theme === 'dark'

  const orderNo = searchParams.get('order_no') || ''
  const [order, setOrder] = useState<OrderData | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!orderNo) {
      setLoading(false)
      return
    }
    getOrder(orderNo)
      .then(res => {
        if (res.data.code === 0) setOrder(res.data.data)
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [orderNo])

  const handleBack = () => {
    const params = new URLSearchParams()
    for (const [k, v] of searchParams.entries()) {
      if (k !== 'order_no' && k !== 'status') params.set(k, v)
    }
    navigate(`/pay?${params.toString()}`)
  }

  const status = order?.status || searchParams.get('status') || 'unknown'
  const isSuccess = status === 'completed' || status === 'paid' || status === 'recharging'

  return (
    <div className={`min-h-screen transition-colors ${isDark ? 'bg-gray-900 text-gray-100' : 'bg-gray-50 text-gray-900'}`}>
      <div className="max-w-md mx-auto px-4 py-16 text-center">
        {loading ? (
          <p className={isDark ? 'text-gray-400' : 'text-gray-500'}>加载中...</p>
        ) : (
          <>
            <div className="text-6xl mb-6">
              {isSuccess ? (
                <span className="text-green-500">&#10003;</span>
              ) : (
                <span className="text-red-500">&#10007;</span>
              )}
            </div>

            <h1 className="text-2xl font-bold mb-2">
              {isSuccess ? '充值成功' : '充值失败'}
            </h1>

            {order && (
              <div className={`mt-6 rounded-xl p-4 text-left text-sm space-y-2
                ${isDark ? 'bg-gray-800' : 'bg-white shadow-sm'}`}>
                <div className="flex justify-between">
                  <span className={isDark ? 'text-gray-400' : 'text-gray-500'}>订单号</span>
                  <span className="font-mono">{order.order_no}</span>
                </div>
                <div className="flex justify-between">
                  <span className={isDark ? 'text-gray-400' : 'text-gray-500'}>金额</span>
                  <span className="font-bold">¥{order.amount.toFixed(2)}</span>
                </div>
                <div className="flex justify-between">
                  <span className={isDark ? 'text-gray-400' : 'text-gray-500'}>状态</span>
                  <StatusBadge status={order.status} />
                </div>
                {order.paid_at && (
                  <div className="flex justify-between">
                    <span className={isDark ? 'text-gray-400' : 'text-gray-500'}>支付时间</span>
                    <span>{new Date(order.paid_at).toLocaleString()}</span>
                  </div>
                )}
              </div>
            )}

            <button
              onClick={handleBack}
              className="mt-8 px-8 py-3 rounded-lg bg-blue-600 text-white font-medium hover:bg-blue-700 transition-colors"
            >
              继续充值
            </button>
          </>
        )}
      </div>
    </div>
  )
}

function StatusBadge({ status }: { status: string }) {
  const map: Record<string, { label: string; color: string }> = {
    pending:    { label: '待支付', color: 'bg-yellow-100 text-yellow-700' },
    paid:       { label: '已支付', color: 'bg-blue-100 text-blue-700' },
    recharging: { label: '充值中', color: 'bg-blue-100 text-blue-700' },
    completed:  { label: '已完成', color: 'bg-green-100 text-green-700' },
    failed:     { label: '失败',   color: 'bg-red-100 text-red-700' },
    expired:    { label: '已过期', color: 'bg-gray-100 text-gray-600' },
  }
  const { label, color } = map[status] ?? { label: status, color: 'bg-gray-100 text-gray-600' }

  return <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${color}`}>{label}</span>
}
