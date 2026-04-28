import { useSettingsStore } from '@/features/settings/settingsStore.ts'
import { computed, h, ref, type VNodeChild } from 'vue'
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
import { summariseDetail } from './notificationContent'

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
  content?: string | (() => VNodeChild)
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
  function renderContent(option: Notification): (() => VNodeChild) | undefined {
    const detail = option.detail
    if (!detail) {
      return undefined
    }

    const summary = summariseDetail(detail)
    const expanded = ref(false)
    const friendly = typeof option.content === 'string' ? option.content : ''

    return () => {
      const showFull = expanded.value || !summary.truncated
      const detailText = showFull ? detail : summary.head + '…'

      const detailBlock = h(
        'pre',
        {
          style:
            'margin: 6px 0 0 0; padding: 6px 8px; font-size: 12px; line-height: 1.4; ' +
            'white-space: pre-wrap; word-break: break-word; user-select: text; cursor: text; ' +
            'opacity: 0.75; background: rgba(127,127,127,0.08); border-radius: 4px; max-height: 200px; overflow: auto;',
        },
        detailText,
      )

      const toggle = summary.truncated
        ? h(
            NButton,
            {
              text: true,
              size: 'tiny',
              type: 'primary',
              style: 'margin-top: 4px;',
              onClick: () => {
                expanded.value = !expanded.value
              },
            },
            {
              default: () =>
                expanded.value ? i18nGlobal.t('common.showLess') : i18nGlobal.t('common.showMore'),
            },
          )
        : null

      return h('div', null, [
        h('div', null, friendly),
        detailBlock,
        toggle,
      ])
    }
  }

  function renderCopyAction(option: Notification): (() => VNodeChild) | undefined {
    const detail = option.detail
    if (!detail) {
      return undefined
    }
    return () =>
      h(
        NButton,
        {
          text: true,
          type: 'primary',
          size: 'small',
          onClick: () => {
            navigator.clipboard.writeText(detail).then(() => {
              window.$messager?.success(i18nGlobal.t('common.copied'))
            })
          },
        },
        { default: () => i18nGlobal.t('common.copy') },
      )
  }

  function applyDetail(option: Notification): void {
    const renderedContent = renderContent(option)
    if (renderedContent) {
      option.content = renderedContent
    }
    const action = renderCopyAction(option)
    if (action) {
      option.action = action
    }
  }

  return {
    show(option: NotificationOptions) {
      return notification.create(option)
    },
    error: (content, option = {}) => {
      option.content = content
      option.title = option.title || i18nGlobal.t('common.error')
      applyDetail(option)
      return notification.error(option)
    },
    info: (content, option = {}) => {
      option.content = content
      applyDetail(option)
      return notification.info(option)
    },
    success: (content, option = {}) => {
      option.content = content
      option.title = option.title || i18nGlobal.t('common.success')
      applyDetail(option)
      return notification.success(option)
    },
    warning: (content, option = {}) => {
      option.content = content
      option.title = option.title || i18nGlobal.t('common.warning')
      applyDetail(option)
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
