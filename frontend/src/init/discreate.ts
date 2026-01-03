import { useSettingsStore } from '@/features/settings/settingsStore.ts'
import { computed, type VNodeChild } from 'vue'
import { createDiscreteApi, darkTheme, type DialogOptions, type MessageOptions, type NotificationReactive } from 'naive-ui'
import { type MessageApiInjection  } from 'naive-ui/lib/message/src/MessageProvider'
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
  const {message, dialog, notification } = createDiscreteApi(['message', 'notification', 'dialog'], {
    configProviderProps: settingsProviderProps,
    messageProviderProps: {
      placement: 'bottom',
      keepAliveOnHover: true,
      containerStyle: {
        marginBottom: '38px',
      },
      themeOverrides: settingsStore.isDark ? darkThemeOverrides.Message : themeOverrides.Message
    },
    notificationProviderProps: {
      max: 5,
      placement: 'bottom-right',
      keepAliveOnHover: true,
      containerStyle: {
        marginBottom: '38px'
      }
    },
  })
  window.$messager = createMessager(message)
  window.$notifier = createNotifier(notification)
  window.$dialoger = createDialoger(dialog)
}

type ContentType = string | (() => VNodeChild)

export interface Messager {
  error: (content: ContentType, options?: MessageOptions | undefined) => void;
  info: (content: ContentType, options?: MessageOptions | undefined) => void;
  loading: (content: ContentType, options?: MessageOptions) => void;
  success: (content: ContentType, options?: MessageOptions | undefined) => void;
  warning: (content: ContentType, options?: MessageOptions | undefined) => void;
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
  duration? : number,
  title?: string,
  meta?: string,
  content?: string,
  icon?: string | (() => VNodeChild),
  action?: string | (() => VNodeChild),
}

export interface Notifier {
  show(option: NotificationOptions): NotificationReactive;
  error: (content: string, option?: Notification) => void;
  info: (content: string, option?: Notification) => void;
  success: (content: string, option?: Notification) => void;
  warning: (content: string, option?: Notification) => void;
}

function createNotifier(notification: NotificationApiInjection) : Notifier {
  return {
    show(option: NotificationOptions) {
      return notification.create(option)
    },
    error: (content, option = {}) => {
      option.content = content
      option.title = option.title || i18nGlobal.t('common.error')
      return notification.error(option)
    },
    info: (content, option = {}) => {
      option.content = content
      return notification.info(option)
    },
    success: (content, option = {}) => {
      option.content = content
      option.title = option.title || i18nGlobal.t('common.success')
      return notification.success(option)
    },
    warning: (content, option = {}) => {
      option.content = content
      option.title = option.title || i18nGlobal.t('common.warning')
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
          onConfirm && onConfirm()
        },
      })
    }
  }
}
