import { describe, expect, it, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

vi.mock('wailsjs/go/api/SettingsProxy', () => ({
  GetSettings: vi.fn(),
  SetSettings: vi.fn(),
  ResetSettings: vi.fn(),
  GetAvailableFonts: vi.fn(),
}))

vi.mock('naive-ui', () => ({
  useOsTheme: () => ({ value: 'light' }),
}))

import { useSettingsStore } from './settingsStore'
import * as settingsProxy from 'wailsjs/go/api/SettingsProxy'

describe('settingsStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('seeds default query and confirmDestructive values', () => {
    const store = useSettingsStore()
    expect(store.query.defaultLimit).toBe(42)
    expect(store.query.defaultPageSize).toBe(25)
    expect(store.query.queryEngine).toBe('builtin')
    expect(store.general.confirmDestructive).toBe(true)
  })

  it('migrates legacy editor.queryEngine when payload lacks query block', async () => {
    ;(settingsProxy.GetSettings as unknown as ReturnType<typeof vi.fn>).mockResolvedValue({
      isSuccess: true,
      data: {
        general: { theme: 'auto', language: 'auto', font: { size: 14 } },
        editor: { font: { size: 14 }, queryEngine: 'mongosh' },
        terminal: { font: { size: 14 }, cursorStyle: 'block' },
        workspaces: { fileExtensions: ['.js'] },
        logging: { level: 'info', consoleEnabled: false, fileEnabled: true, maxSizeMB: 10, maxBackups: 5 },
      },
    })
    const store = useSettingsStore()
    await store.loadSettings()
    expect(store.query.queryEngine).toBe('mongosh')
    expect(store.query.defaultLimit).toBe(42)
    expect(store.query.defaultPageSize).toBe(25)
  })

  it('defaults confirmDestructive to true when missing from payload', async () => {
    ;(settingsProxy.GetSettings as unknown as ReturnType<typeof vi.fn>).mockResolvedValue({
      isSuccess: true,
      data: {
        general: { theme: 'auto', language: 'auto', font: { size: 14 } },
        editor: { font: { size: 14 } },
        query: { defaultLimit: 100, defaultPageSize: 50, queryEngine: 'builtin' },
        terminal: { font: { size: 14 }, cursorStyle: 'block' },
        workspaces: { fileExtensions: ['.js'] },
        logging: { level: 'info', consoleEnabled: false, fileEnabled: true, maxSizeMB: 10, maxBackups: 5 },
      },
    })
    const store = useSettingsStore()
    await store.loadSettings()
    expect(store.general.confirmDestructive).toBe(true)
    expect(store.query.defaultLimit).toBe(100)
    expect(store.query.defaultPageSize).toBe(50)
  })
})
