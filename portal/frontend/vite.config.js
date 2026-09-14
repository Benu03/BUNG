import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// The portal is served at the site root by nginx (see /nginx/nginx.conf),
// unlike modules which live under /<module>/.
export default defineConfig({
  plugins: [react()],
  base: '/',
  server: {
    host: true,
    port: 5173,
  },
})
