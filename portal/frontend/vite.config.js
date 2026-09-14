import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// The portal is served at the site root by nginx (see /nginx/nginx.conf),
// unlike modules which live under /<module>/.
export default defineConfig({
  plugins: [react(), tailwindcss()],
  base: '/',
  server: {
    host: true,
    port: 5173,
  },
})
