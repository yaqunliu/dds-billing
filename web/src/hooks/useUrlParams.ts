import { useMemo } from 'react'
import { useSearchParams } from 'react-router-dom'

export interface UrlParams {
  user_id: string
  token: string
  theme: 'light' | 'dark'
  lang: string
  ui_mode: 'standalone' | 'embedded'
}

export function useUrlParams(): UrlParams {
  const [searchParams] = useSearchParams()

  return useMemo(() => ({
    user_id: searchParams.get('user_id') || '',
    token: searchParams.get('token') || '',
    theme: (searchParams.get('theme') as 'light' | 'dark') || 'light',
    lang: searchParams.get('lang') || 'zh',
    ui_mode: (searchParams.get('ui_mode') as 'standalone' | 'embedded') || 'standalone',
  }), [searchParams])
}
