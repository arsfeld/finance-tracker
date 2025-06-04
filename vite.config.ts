import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { resolve } from 'path'

export default defineConfig({
  plugins: [react()],
  root: 'resources',
  base: '/',
  resolve: {
    alias: {
      '@': resolve(__dirname, 'resources/js'),
    },
  },
  build: {
    manifest: true,
    outDir: '../src/web/static/build',
    emptyOutDir: true,
    rollupOptions: {
      input: {
        app: resolve(__dirname, 'resources/js/app.tsx'),
      },
    },
  },
  server: {
    host: '0.0.0.0', // Bind to all interfaces so it can be accessed from dev.local
    port: 5173,
    strictPort: true,
    cors: {
      origin: ['http://dev.local:8080', 'http://localhost:8080'],
      credentials: true,
    }, // Enable CORS for cross-origin requests
    hmr: {
      host: 'dev.local', // Use your actual development domain
    },
  },
})