import dayjs from 'dayjs'
import duration from 'dayjs/plugin/duration'
import relativeTime from 'dayjs/plugin/relativeTime'
import { createApp, nextTick } from 'vue'
import App from './app/App.vue'
import { createPinia } from 'pinia'
import { loadEnvironment } from './init/environment'
import { initMonaco } from './init/monaco'
import { initCharts } from './init/charts'
import { initDiscreteApi } from './init/discreate'
import { useNotifier } from '@/utils/dialog'
import { i18n } from '@/i18n'
import './css/app.scss'

dayjs.extend(duration)
dayjs.extend(relativeTime)

async function initApp() {
  const app = createApp(App)
  app.use(i18n)
  app.use(createPinia())

  await loadEnvironment()
  initMonaco()
  initCharts()

  await initDiscreteApi()
  app.config.errorHandler = (err, instance, info) => {
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
