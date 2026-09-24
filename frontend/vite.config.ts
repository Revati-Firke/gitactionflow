import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      // Same-origin proxy keeps session cookies simple during local dev.
      '/api': 'http://127.0.0.1:8080',
      '/auth': 'http://127.0.0.1:8080',
    },
  },
})
