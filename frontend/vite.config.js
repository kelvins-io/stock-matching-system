import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    host: '0.0.0.0',
    port: 6173,
    proxy: {
      '/api': {
        target: process.env.VITE_API_PROXY || 'http://localhost:10020',
        changeOrigin: true
      }
    }
  },
  preview: {
    host: '0.0.0.0',
    port: 6173
  }
})
