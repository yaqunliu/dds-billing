import { useEffect } from 'react'
import { BrowserRouter, Routes, Route, useSearchParams } from 'react-router-dom'
import Pay from './pages/Pay'
import Result from './pages/Result'
import Orders from './pages/Orders'

function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [searchParams] = useSearchParams()
  const theme = searchParams.get('theme') || 'light'
  const uiMode = searchParams.get('ui_mode') || 'standalone'

  useEffect(() => {
    // Apply dark class for Tailwind dark mode if needed
    document.documentElement.setAttribute('data-theme', theme)
    document.documentElement.setAttribute('data-ui-mode', uiMode)

    if (theme === 'dark') {
      document.documentElement.classList.add('dark')
    } else {
      document.documentElement.classList.remove('dark')
    }
  }, [theme, uiMode])

  return <>{children}</>
}

function App() {
  return (
    <BrowserRouter>
      <ThemeProvider>
        <Routes>
          <Route path="/pay" element={<Pay />} />
          <Route path="/pay/result" element={<Result />} />
          <Route path="/pay/orders" element={<Orders />} />
          <Route path="*" element={<Pay />} />
        </Routes>
      </ThemeProvider>
    </BrowserRouter>
  )
}

export default App
