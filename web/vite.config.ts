import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// SailGuard web port map: docs/internal/ports.md
export default defineConfig({
  plugins: [react()],
  server: {
    host: '127.0.0.1',
    port: 15180,
    strictPort: true,
    proxy: {
      '/v1': {
        target: 'http://127.0.0.1:18080',
        changeOrigin: true,
      },
    },
  },
})
