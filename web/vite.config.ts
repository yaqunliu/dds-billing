import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  base: '/pay/',
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      '/pay/api': {
        target: 'http://localhost:3000',
        rewrite: (path) => path.replace(/^\/pay\/api/, '/api'),
      },
      '/health': 'http://localhost:3000',
    },
  },
})
