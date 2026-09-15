import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// base must match the nginx location prefix this module is served under
// (see /modules/chat/nginx.conf). Override at build time with the
// VITE_BASE env var if you duplicate this module under a new name.
export default defineConfig({
  plugins: [react(), tailwindcss()],
  base: process.env.VITE_BASE || '/chat/',
  server: {
    host: true,
    port: 5173,
  },
})
