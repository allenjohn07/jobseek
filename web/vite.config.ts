import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import tailwindcss from '@tailwindcss/vite'
import path from 'node:path'

export default defineConfig({
  plugins: [svelte(), tailwindcss()],  // compile .svelte files + generate Tailwind CSS
  resolve: {
    alias: { $lib: path.resolve(__dirname, './src/lib') },
  },
  server: {
    proxy: {
      '/api': 'http://localhost:8080', // forward /api/* to the Go server during development
    },
  },
})