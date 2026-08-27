import { defineConfig } from 'vitest/config'
import { fileURLToPath, URL } from 'node:url'
import vue from '@vitejs/plugin-vue'
import AutoImport from 'unplugin-auto-import/vite'
import { NaiveUiResolver } from 'unplugin-vue-components/resolvers'
import Components from 'unplugin-vue-components/vite'

export default defineConfig({
  // Mirrors the app's auto-imports so component tests can mount SFCs that rely
  // on bare `useDialog()` and unimported Naive UI components.
  plugins: [
    vue(),
    AutoImport({
      imports: [{ 'naive-ui': ['useDialog', 'useMessage', 'useNotification', 'useLoadingBar'] }],
    }),
    Components({ resolvers: [NaiveUiResolver()] }),
  ],
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
        // Lowered when the fully-covered buildInfoStore was deleted along
        // with the Microsoft Store distribution; no tests were removed.
        //
        // Lowered again when the Monaco completion harness and the QueryTab
        // remount tests landed: v8 only reports files a test actually loads, so
        // mounting QueryTab.vue and importing useMonacoCompletions.ts pulled
        // ~1200 largely uncovered lines into the denominator for the first
        // time. Tests were added, not removed — the measured surface grew.
        statements: 52.3,
        branches: 48.24,
        functions: 43.89,
        lines: 52.6,
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
