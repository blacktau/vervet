import dayjs from 'dayjs'
import duration from 'dayjs/plugin/duration'
import relativeTime from 'dayjs/plugin/relativeTime'
import { createApp, nextTick } from 'vue'
import App from './app/App.vue'
import { createPinia } from 'pinia'
import { loadEnvironment } from './init/environment'
import { initMonaco } from './init/monaco'
import { initCharts } from './init/charts'
import { initDiscreteApi } from './init/discrete'
import { useNotifier } from '@/utils/dialog'
import { i18n } from '@/i18n'
import './css/app.scss'
import '@/utils/logging'

dayjs.extend(duration)
dayjs.extend(relativeTime)

// Workaround for WebKit2GTK on Linux: Ctrl+V is intercepted by GTK
// before reaching the webview. We read from the clipboard API and
// use execCommand to insert the text, which correctly replaces any selection.
function initClipboardWorkaround() {
  document.addEventListener('keydown', async (e) => {
    if (e.key === 'v' && (e.ctrlKey || e.metaKey) && !e.shiftKey && !e.altKey) {
      try {
        const text = await navigator.clipboard.readText()
        if (text) {
          document.execCommand('insertText', false, text)
          e.preventDefault()
        }
      } catch {
        // Clipboard API not available or denied
      }
    }
  })
}

async function initApp() {
  const app = createApp(App)
  app.use(i18n)
  app.use(createPinia())

  await loadEnvironment()
  initMonaco()
  initCharts()
  initClipboardWorkaround()

  await initDiscreteApi()
  app.config.errorHandler = (err) => {
    nextTick().then(() => {
      try {
        const error = err as Error
        const content = error.toString()
        const notifier = useNotifier()
        notifier.error(content, {
          title: i18n.global.t('common.error'),
          meta: 'Please see console for details',
        })
        console.error(err)
      } catch (e) {
        console.error(e)
      }
    })
  }

  app.mount('#app')
}

initApp()
