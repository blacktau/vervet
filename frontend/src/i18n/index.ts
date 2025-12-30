import enGB from './en-GB'
import { createI18n } from 'vue-i18n'

export const translations = {
  'en-GB': enGB,
}

type MessageSchema = typeof enGB

export const i18n = createI18n<[MessageSchema], string>({
  legacy: false,
  locale: 'en-GB',
  globalInjection: true,
  messages: {
    ...translations,
  }
})

export const i18nGlobal = i18n.global
