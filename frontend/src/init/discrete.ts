import { useSettingsStore } from '@/features/settings/settingsStore.ts'
import { computed, h, type VNodeChild } from 'vue'
import {
  createDiscreteApi,
  darkTheme,
  NButton,
  type DialogOptions,
  type MessageOptions,
  type NotificationReactive
} from 'naive-ui'
import { type MessageApiInjection } from 'naive-ui/lib/message/src/MessageProvider'
import { type NotificationApiInjection } from 'naive-ui/es/notification/src/NotificationProvider'
import { type NotificationOptions } from 'naive-ui/es/notification'
import { darkThemeOverrides, themeOverrides } from '@/utils/theme'
import { i18nGlobal } from '@/i18n'
import { type DialogApiInjection } from 'naive-ui/es/dialog/src/DialogProvider'
import { type DialogReactive } from 'naive-ui/lib'

export async function initDiscreteApi() {
  const settingsStore = useSettingsStore()
  const settingsProviderProps = computed(() => ({
    theme: settingsStore.isDark ? darkTheme : undefined,
    themeOverrides,
  }))
  const { message, dialog, notification } = createDiscreteApi(
    ['message', 'notification', 'dialog'],
    {
      configProviderProps: settingsProviderProps,
      messageProviderProps: {
        placement: 'bottom',
        keepAliveOnHover: true,
        containerStyle: {
          marginBottom: '38px',
        },
        themeOverrides: settingsStore.isDark ? darkThemeOverrides.Message : themeOverrides.Message,
      },
      notificationProviderProps: {
        max: 5,
        placement: 'bottom-right',
        keepAliveOnHover: true,
        containerStyle: {
          marginBottom: '38px',
        },
      },
    },
  )
  window.$messager = createMessager(message)
  window.$notifier = createNotifier(notification)
  window.$dialoger = createDialoger(dialog)
}

type ContentType = string | (() => VNodeChild)

export interface Messager {
  error: (content: ContentType, options?: MessageOptions | undefined) => void
  info: (content: ContentType, options?: MessageOptions | undefined) => void
  loading: (content: ContentType, options?: MessageOptions) => void
  success: (content: ContentType, options?: MessageOptions | undefined) => void
  warning: (content: ContentType, options?: MessageOptions | undefined) => void
}

function createMessager(message: MessageApiInjection): Messager {
  return {
    error: (content: ContentType, options: MessageOptions | undefined = undefined) => {
      return message.error(content, options)
    },
    info: (content: ContentType, options: MessageOptions | undefined = undefined) => {
      return message.info(content, options)
    },
    loading: (content: ContentType, options: MessageOptions = {}) => {
      options.duration = options.duration != null ? options.duration : 30000
      options.keepAliveOnHover =
        options.keepAliveOnHover !== undefined ? options.keepAliveOnHover : true
      return message.loading(content, options)
    },
    success: (content: ContentType, options: MessageOptions | undefined = undefined) => {
      return message.success(content, options)
    },
    warning: (content: ContentType, options: MessageOptions | undefined = undefined) => {
      return message.warning(content, options)
    },
  }
}

interface Notification {
  duration?: number
  title?: string
  meta?: string
  content?: string
  detail?: string
  icon?: string | (() => VNodeChild)
  action?: string | (() => VNodeChild)
}

export interface Notifier {
  error: (content: string, option?: Notification) => void
  info: (content: string, option?: Notification) => void
  success: (content: string, option?: Notification) => void
  warning: (content: string, option?: Notification) => void

  show(option: NotificationOptions): NotificationReactive
}

function createNotifier(notification: NotificationApiInjection): Notifier {
  function withDetailAction(option: Notification & { content?: string }): void {
    if (option.detail) {
      const detail = option.detail
      option.action = () =>
        h(
          NButton,
          {
            text: true,
            type: 'primary',
            size: 'small',
            onClick: () => {
              window.$dialoger.show({
                title: option.title || i18nGlobal.t('common.error'),
                content: () =>
                  h('pre', {
                    style: 'white-space: pre-wrap; word-break: break-all; margin: 0; font-size: 13px; user-select: text; cursor: text;',
                  }, detail),
                positiveText: i18nGlobal.t('common.copyToClipboard'),
                onPositiveClick: () => {
                  navigator.clipboard.writeText(detail)
                },
              })
            },
          },
          { default: () => i18nGlobal.t('common.details') },
        )
    }
  }

  return {
    show(option: NotificationOptions) {
      return notification.create(option)
    },
    error: (content, option = {}) => {
      option.content = content
      option.title = option.title || i18nGlobal.t('common.error')
      withDetailAction(option)
      return notification.error(option)
    },
    info: (content, option = {}) => {
      option.content = content
      withDetailAction(option)
      return notification.info(option)
    },
    success: (content, option = {}) => {
      option.content = content
      option.title = option.title || i18nGlobal.t('common.success')
      withDetailAction(option)
      return notification.success(option)
    },
    warning: (content, option = {}) => {
      option.content = content
      option.title = option.title || i18nGlobal.t('common.warning')
      withDetailAction(option)
      return notification.warning(option)
    },
  }
}

export interface Dialoger {
  show(options: DialogOptions): DialogReactive
  warning(content: string, onConfirm?: () => void): DialogReactive
}

function createDialoger(dialog: DialogApiInjection) {
  return {
    show: (options: DialogOptions) => {
      options.closable = options.closable === true
      options.autoFocus = options.autoFocus === true
      options.transformOrigin = 'center'
      return dialog.create(options)
    },
    warning: (content: string, onConfirm?: () => void) => {
      return dialog.warning({
        title: i18nGlobal.t('common.warning'),
        content: content,
        closable: false,
        autoFocus: false,
        transformOrigin: 'center',
        positiveText: i18nGlobal.t('common.confirm'),
        negativeText: i18nGlobal.t('common.cancel'),
        onPositiveClick: () => {
          if (onConfirm) {
            onConfirm()
          }
        },
      })
    },
  }
}
