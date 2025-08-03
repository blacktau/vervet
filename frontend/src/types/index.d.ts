export {};

import { Dialoger, Messager, Notifier } from '@/init/discrete'

declare global {
  interface Window {
    '$messager': Messager,
    '$notifier': Notifier,
    '$dialoger': Dialoger,
    '$t': (key: string) => string
  }
}
