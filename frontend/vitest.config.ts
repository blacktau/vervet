import { defineConfig } from 'vitest/config'
import { fileURLToPath, URL } from 'node:url'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  test: {
    globals: true,
    environment: 'node',
    include: ['src/**/*.test.ts'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'lcov'],
      // Generated bindings, static string tables and icon components carry no
      // logic. Measuring them inflates the number and teaches us to ignore it.
      exclude: [
        'src/**/*.d.ts',
        'src/i18n/**',
        'src/features/icon/**',
        'wailsjs/**',
      ],
      thresholds: {
        // These are a ratchet, not a target. `autoUpdate` raises them when
        // coverage improves and never lowers them. Disabled under CI so a
        // build can't rewrite its own gate.
        autoUpdate: !process.env.CI,
        statements: 58.01,
        branches: 53.08,
        functions: 55.48,
        lines: 58.74,
      },
    },
  },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      wailsjs: fileURLToPath(new URL('./wailsjs', import.meta.url)),
    },
  },
})
