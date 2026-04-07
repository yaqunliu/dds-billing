import axios from 'axios'

const api = axios.create({
  baseURL: '/api',
  timeout: 10000,
})

export interface OrderCreateRequest {
  token: string
  amount: number
  payment_type: 'wxpay' | 'alipay'
}

export interface OrderData {
  order_no: string
  amount: number
  status: string
  qr_code_url: string
  pay_url: string
  expires_at: string
  paid_at?: string
}

export interface ApiResponse<T> {
  code: number
  message?: string
  data: T
}

export interface AppConfig {
  enabled_types: string[]
  min_amount: number
  max_amount: number
}

export const getConfig = () => api.get<ApiResponse<AppConfig>>('/config')

export const createOrder = (data: OrderCreateRequest) =>
  api.post<ApiResponse<OrderData>>('/orders', data)

export const getOrder = (orderNo: string) =>
  api.get<ApiResponse<OrderData>>(`/orders/${orderNo}`)

export const getOrders = (token: string, page = 1, pageSize = 20) =>
  api.get<ApiResponse<{ list: OrderData[]; total: number }>>('/orders', {
    params: { token, page, page_size: pageSize },
  })

export default api
