import dayjs from 'dayjs'
import duration from 'dayjs/plugin/duration'
import relativeTime from 'dayjs/plugin/relativeTime'
import { createApp, nextTick } from 'vue'
import App from './app/App.vue'
import { createPinia } from 'pinia'
import { isLinux, loadEnvironment } from './init/environment'
import { initMonaco } from './init/monaco'
import { initDiscreteApi } from './init/discrete'
import { useNotifier } from '@/utils/dialog'
import { i18n } from '@/i18n'
import './css/app.scss'
import '@/utils/logging'

dayjs.extend(duration)
dayjs.extend(relativeTime)

// Workaround for WebKitGTK on Linux: Ctrl+V is intercepted by GTK
// before reaching the webview. Still needed on WebKitGTK 6.0 (checked
// 2026-11-09: without it nothing pastes, with it the text pastes once). We read from the clipboard API and
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
  if (isLinux()) {
    initClipboardWorkaround()
  }

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

// If startup fails there is nothing else on screen, so show the error rather
// than leaving an empty window.
initApp().catch((err: unknown) => {
  console.error('Vervet failed to start', err)
  const pre = document.createElement('pre')
  pre.style.cssText = 'margin: 16px; white-space: pre-wrap; font: 13px monospace; color: #b00020'
  pre.textContent = `Vervet failed to start:\n\n${err instanceof Error ? (err.stack ?? err.message) : String(err)}`
  document.body.replaceChildren(pre)
})
