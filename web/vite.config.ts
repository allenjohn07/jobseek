import { defineConfig } from 'vitest/config'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import tailwindcss from '@tailwindcss/vite'
import path from 'node:path'

export default defineConfig({
  plugins: [svelte(), tailwindcss()],  // compile .svelte files + generate Tailwind CSS
  resolve: {
    // Svelte must resolve to its browser build under jsdom, otherwise mount() is unavailable.
    conditions: process.env.VITEST ? ['browser'] : undefined,
    alias: { $lib: path.resolve(import.meta.dirname, './src/lib') },
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test-setup.ts'],
    include: ['src/**/*.test.ts'],
    restoreMocks: true,
  },
  server: {
    proxy: {
      '/api': 'http://localhost:8080', // forward /api/* to the Go server during development
    },
  },
})