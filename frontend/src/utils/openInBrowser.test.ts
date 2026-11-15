import { beforeEach, describe, expect, it, vi } from 'vitest'

const error = vi.fn()
vi.mock('@/utils/dialog', () => ({ useNotifier: () => ({ error }) }))

const openURL = vi.fn()
vi.mock('bindings/vervet/internal/api/systemproxy', () => ({ OpenURL: (url: string) => openURL(url) }))

import { openInBrowser } from './openInBrowser'

describe('openInBrowser', () => {
  beforeEach(() => {
    error.mockReset()
    openURL.mockReset()
  })

  it('opens the link without a notice', async () => {
    openURL.mockResolvedValue({ isSuccess: true })
    await openInBrowser('https://github.com/blacktau/vervet')
    expect(openURL).toHaveBeenCalledWith('https://github.com/blacktau/vervet')
    expect(error).not.toHaveBeenCalled()
  })

  it('shows the refused link so the user can copy it', async () => {
    openURL.mockResolvedValue({ isSuccess: false, errorDetail: 'refusing to open URL: shell metacharacters not allowed' })
    await openInBrowser('https://host/app;jsessionid=abc')
    expect(error).toHaveBeenCalledWith('https://host/app;jsessionid=abc', {
      title: 'Could not open link',
      detail: 'refusing to open URL: shell metacharacters not allowed',
    })
  })
})
