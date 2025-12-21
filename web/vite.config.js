import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { readFileSync } from 'fs'

// Version aus package.json lesen
const pkg = JSON.parse(readFileSync('./package.json', 'utf-8'))

export default defineConfig({
  plugins: [vue()],
  define: {
    __APP_VERSION__: JSON.stringify(pkg.version),
    __BUILD_DATE__: JSON.stringify(new Date().toISOString().split('T')[0]),
    __BUILD_TIME__: JSON.stringify(new Date().toTimeString().split(' ')[0])
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:2025',
        changeOrigin: true
      }
    }
  },
  build: {
    outDir: '../cmd/navigator/dist',
    emptyOutDir: true
  }
})
