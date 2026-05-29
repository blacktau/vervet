import { useOsTheme } from 'naive-ui'
import { defineStore } from 'pinia'
import { i18nGlobal, translations } from '@/i18n'
import { cloneDeep, get, isEmpty, set, split } from 'lodash'
import * as settingsProxy from 'wailsjs/go/api/SettingsProxy'
import { type models } from 'wailsjs/go/models.ts'
import { useNotifier } from '@/utils/dialog.ts'

// Tracks an in-flight font enumeration so the expensive backend call isn't
// fired twice when the background startup load and a settings-open race.
let fontListInFlight: Promise<void> | null = null

type SettingsStore = models.Settings & {
  fontList: models.Font[]
  fontListLoaded: boolean
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
        confirmDestructive: true,
      },
      editor: {
        font: {
          size: 14,
        },
        showLineNumbers: true,
      },
      query: {
        defaultLimit: 42,
        defaultPageSize: 25,
        queryEngine: 'builtin',
      },
      terminal: {
        font: {
          size: 14,
        },
        cursorStyle: 'block',
      },
      workspaces: {
        fileExtensions: ['.js', '.mongodb'] as string[],
      },
      logging: {
        level: 'info',
        consoleEnabled: false,
        fileEnabled: true,
        maxSizeMB: 10,
        maxBackups: 5,
      },
      fontList: [],
      fontListLoaded: false,
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
      return options
    },
    currentLanguage() {
      let language: string = this.general.language || 'auto'
      if (language === 'auto') {
        const systemLanguage = navigator.language
        language = split(systemLanguage, '-')[0] || 'en'
      }
      return language || 'en'
    },
    fontOptions(state: SettingsStore) {
      return state.fontList
    },
    monoFontOptions(state: SettingsStore) {
      return state.fontList.filter((x) => x.isFixedWidth)
    },
    uiFont(state: SettingsStore) {
      return fontToStyle(state.general.font)
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
    logLevelOptions() {
      return [
        { value: 'debug', label: 'settings.logging.levels.debug' },
        { value: 'info', label: 'settings.logging.levels.info' },
        { value: 'warn', label: 'settings.logging.levels.warn' },
        { value: 'error', label: 'settings.logging.levels.error' },
      ]
    },
    isDark(): boolean {
      const th = this.general.theme || 'auto'
      if (th === 'dark') {
        return true
      }
      if (th === 'light') {
        return false
      }
      return useOsTheme().value === 'dark'
    },
  },
  actions: {
    _applyConfiguration(settings: models.Settings) {
      const data = settings as unknown as Record<string, unknown>
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
      const query = get(result.data, 'query')
      if (query === undefined) {
        const legacyEngine = get(result.data, 'editor.queryEngine') as string | undefined
        set(this, 'query', {
          defaultLimit: 42,
          defaultPageSize: 25,
          queryEngine: legacyEngine ?? 'builtin',
        })
      }
      const confirmDestructive = get(result.data, 'general.confirmDestructive')
      if (confirmDestructive === undefined) {
        set(this, 'general.confirmDestructive', true)
      }
      const fileExtensions = get(result.data, 'workspaces.fileExtensions')
      if (fileExtensions == null) {
        set(this, 'workspaces.fileExtensions', ['.js', '.mongodb'])
      }
      const logging = get(result.data, 'logging')
      if (logging === undefined) {
        set(this, 'logging', {
          level: 'info',
          consoleEnabled: false,
          fileEnabled: true,
          maxSizeMB: 10,
          maxBackups: 5,
        })
      }

      i18nGlobal.locale = this.currentLanguage
    },
    async loadFontList(): Promise<void> {
      if (this.fontListLoaded || fontListInFlight) {
        return fontListInFlight ?? undefined
      }
      fontListInFlight = (async () => {
        try {
          const result = await settingsProxy.GetAvailableFonts()
          if (!result.isSuccess) {
            const notifier = useNotifier()
            notifier.error(i18nGlobal.t(`errors.${result.errorCode}`), { title: i18nGlobal.t('errorTitles.saveSettings'), detail: result.errorDetail })
            return
          }

          this.fontList = result.data
          this.fontListLoaded = true
        } finally {
          fontListInFlight = null
        }
      })()
      return fontListInFlight
    },
    async saveConfiguration(): Promise<boolean> {
      const payload: models.Settings = {
        window: this.window,
        general: this.general,
        editor: this.editor,
        query: this.query,
        terminal: this.terminal,
        workspaces: this.workspaces,
        updates: this.updates,
        logging: this.logging,
      } as models.Settings
      const result = await settingsProxy.SetSettings(payload)
      if (!result.isSuccess) {
        const notifier = useNotifier()
        notifier.error(i18nGlobal.t(`errors.${result.errorCode || 'unknown_error'}`), {
          title: i18nGlobal.t('errorTitles.saveSettings'),
          detail: result.errorDetail,
        })
      }
      return result.isSuccess
    },
    async restoreConfiguration(): Promise<boolean> {
      const result = await settingsProxy.ResetSettings()
      if (result.isSuccess) {
        this._applyConfiguration(result.data)
        if (get(this, 'workspaces.fileExtensions') == null) {
          set(this, 'workspaces.fileExtensions', ['.js', '.mongodb'])
        }
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
    const families = Array.isArray(font.family) ? font.family : [font.family]
    style['fontFamily'] = families.map((f) => `"${f}"`).join(', ')
  }

  if (isEmpty(style['fontFamily']) && !isEmpty(defaultFontFamily)) {
    style['fontFamily'] = defaultFontFamily
  }

  return style
}
