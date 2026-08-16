import DefaultTheme from 'vitepress/theme'
import type { Theme } from 'vitepress'
import DownloadPicker from './DownloadPicker.vue'

export default {
  extends: DefaultTheme,
  enhanceApp({ app }) {
    app.component('DownloadPicker', DownloadPicker)
  },
} satisfies Theme
