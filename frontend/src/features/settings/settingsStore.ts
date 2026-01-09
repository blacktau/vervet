import { useOsTheme } from 'naive-ui'
import { defineStore } from 'pinia'
import { i18nGlobal, translations } from '@/i18n'
import { cloneDeep, get, isEmpty, join, map, set, split } from 'lodash'
import * as settingsProxy from 'wailsjs/go/api/SettingsProxy'
import { type models } from 'wailsjs/go/models.ts'
import { useNotifier } from '@/utils/dialog.ts'

const theme = useOsTheme()

type SettingsStore = models.Settings & {
  fontList: models.Font[]
  previousVersion?: models.Settings
}

export const useSettingsStore = defineStore('settings', {
  state: () =>
    ({
      window: {
        width: 0,
        height: 0,
        asideWidth: 0,
        maximised: false,
      },
      general: {
        theme: 'auto',
        language: 'auto',
        font: {
          size: 14,
        },
      },
      editor: {
        font: {
          size: 14,
        },
        showLineNumbers: true,
      },
      terminal: {
        font: {
          size: 14,
        },
        cursorStyle: 'block',
      },
      fontList: [],
    }) as unknown as SettingsStore,
  getters: {
    themeOptions() {
      return [
        {
          value: 'light',
          label: 'settings.general.themeLight',
        },
        {
          value: 'dark',
          label: 'settings.general.themeDark',
        },
        {
          value: 'auto',
          label: 'settings.general.themeAuto',
        },
      ]
    },
    languageOptions() {
      const options = Object.entries(translations).map(([key, value]) => ({
        value: key,
        label: value['name'],
      }))
      options.splice(0, 0, {
        value: 'auto',
        label: i18nGlobal.t('settings.general.systemLanguage'),
      })
    },
    currentLanguage() {
      let language: string = this.general.language || 'auto'
      if (language === 'auto') {
        const systemLanguage = navigator.language
        language = split(systemLanguage, '-')[0] || 'en'
      }
      return language || 'en'
    },
    showLineNum(state: SettingsStore) {
      return get(state.editor, 'showLineNum', true)
    },
    fontOptions(state: SettingsStore) {
      return state.fontList
    },
    monoFontOptions(state: SettingsStore) {
      return state.fontList.filter((x) => x.isFixedWidth)
    },
    uiFont(state: SettingsStore) {
      return fontToStyle(state.general.font, 'monaco')
    },
    terminalFont(state: SettingsStore) {
      return fontToStyle(state.terminal.font, 'Courier New')
    },
    terminalCursorOptions() {
      return [
        {
          value: 'block',
          label: 'settings.terminal.cursorStyleBlock',
        },
        {
          value: 'underline',
          label: 'settings.terminal.cursorStyleUnderline',
        },
        {
          value: 'bar',
          label: 'settings.terminal.cursorStyleBar',
        },
      ]
    },
    isDark(): boolean {
      const th = this.general.theme || 'auto'
      return th === 'dark' || (th === 'auto' && theme.value === 'dark')
    },
  },
  actions: {
    _applyConfiguration(data: Record<string, any>) {
      for (const key in data) {
        set(this, key, data[key])
      }
    },
    async loadSettings(): Promise<void> {
      const result = await settingsProxy.GetSettings()
      if (!result.isSuccess) {
        return
      }

      this.previousVersion = cloneDeep(result.data)
      this._applyConfiguration(result.data)
      const showLineNum = get(result.data, 'editor.showLineNum')
      if (showLineNum === undefined) {
        set(result.data, 'editor.showLineNum', true)
      }

      const showFolding = get(result.data, 'editor.showFolding')
      if (showFolding === undefined) {
        set(result.data, 'editor.showFolding', true)
      }
      const dropText = get(result.data, 'editor.dropText')
      if (dropText === undefined) {
        set(result.data, 'editor.dropText', true)
      }
      const links = get(result.data, 'editor.links')
      if (links === undefined) {
        set(result.data, 'editor.links', true)
      }
      i18nGlobal.locale = this.currentLanguage
    },
    async loadFontList(): Promise<void> {
      const result = await settingsProxy.GetAvailableFonts()
      if (!result.isSuccess) {
        const notifier = useNotifier()
        notifier.error(`error retrieving available fonts: ${result.error}`)
        return
      }

      this.fontList = result.data
    },
    async saveConfiguration(): Promise<boolean> {
      const result = await settingsProxy.SetSettings(this)
      return result.isSuccess
    },
    async restoreConfiguration(): Promise<boolean> {
      const result = await settingsProxy.ResetSettings()
      if (result.isSuccess) {
        this._applyConfiguration(result.data)
        return true
      }
      return false
    },
  },
})

function fontToStyle(font: models.FontSettings, defaultFontFamily?: string) {
  const style: Record<string, string | undefined> = {
    fontSize: (font.size || 14) + 'px',
  }
  if (!isEmpty(font.family)) {
    style['fontFamily'] = join(
      map(font.family, (f) => `"${f}`),
      ',',
    )
  }

  if (isEmpty(style['fontFamily']) && !isEmpty(defaultFontFamily)) {
    style['fontFamily'] = defaultFontFamily
  }

  return style
}
