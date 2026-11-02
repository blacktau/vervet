import { describe, expect, it, vi } from 'vitest'
import { i18nGlobal } from '@/i18n'
import {
  errorCodeText,
  notifyResultError,
  resultDetailText,
  resultErrorLines,
  resultErrorText,
} from './resultError'

const notifierError = vi.hoisted(() => vi.fn())
vi.mock('@/utils/dialog', () => ({ useNotifier: () => ({ error: notifierError }) }))

const generic = () => i18nGlobal.t('errors.unknown_error')
const notAuthorised = () => i18nGlobal.t('errors.not_authorized')

describe('errorCodeText', () => {
  it('translates a code', () => {
    expect(errorCodeText('not_authorized')).toBe(notAuthorised())
  })

  it('uses the generic message when there is no code', () => {
    expect(errorCodeText(undefined)).toBe(generic())
    expect(errorCodeText('')).toBe(generic())
  })

  it('uses the generic message for a code with no translation', () => {
    expect(errorCodeText('made_up')).toBe(generic())
  })
})

describe('resultErrorText', () => {
  it('prefers the translation of a known code', () => {
    expect(resultErrorText({ errorCode: 'not_authorized', errorDetail: 'raw' })).toBe(notAuthorised())
  })

  it("shows the detail for the Go side's catch-all code", () => {
    expect(resultErrorText({ errorCode: 'unknown_error', errorDetail: 'E11000 duplicate key' })).toBe(
      'E11000 duplicate key',
    )
  })

  it('shows the detail when there is no code', () => {
    expect(resultErrorText({ errorDetail: 'E11000 duplicate key' })).toBe('E11000 duplicate key')
  })

  it('shows the detail when the code has no translation', () => {
    expect(resultErrorText({ errorCode: 'made_up', errorDetail: 'raw' })).toBe('raw')
  })

  it('never shows a raw code', () => {
    expect(resultErrorText({ errorCode: 'unknown_error' })).toBe(generic())
    expect(resultErrorText({ errorCode: 'made_up' })).toBe(generic())
  })

  it('uses the generic message when the result says nothing', () => {
    expect(resultErrorText({})).toBe(generic())
  })
})

describe('resultErrorLines', () => {
  it('keeps the detail beneath a known code', () => {
    expect(resultErrorLines({ errorCode: 'not_authorized', errorDetail: 'raw' })).toBe(`${notAuthorised()}\nraw`)
  })

  it('does not repeat a detail that matches the headline', () => {
    expect(resultErrorLines({ errorCode: 'not_authorized', errorDetail: notAuthorised() })).toBe(notAuthorised())
  })

  it("shows only the detail for the Go side's catch-all code", () => {
    expect(resultErrorLines({ errorCode: 'unknown_error', errorDetail: 'raw' })).toBe('raw')
  })

  it('uses the headline or generic message when there is no detail', () => {
    expect(resultErrorLines({ errorCode: 'not_authorized' })).toBe(notAuthorised())
    expect(resultErrorLines({})).toBe(generic())
  })
})

describe('resultDetailText', () => {
  it('prefers the detail over a known code', () => {
    expect(resultDetailText({ errorCode: 'connection_failed', errorDetail: 'open x: permission denied' })).toBe(
      'open x: permission denied',
    )
  })

  it('falls back to the headline, then the generic message', () => {
    expect(resultDetailText({ errorCode: 'not_authorized' })).toBe(notAuthorised())
    expect(resultDetailText({})).toBe(generic())
  })
})

describe('notifyResultError', () => {
  it('passes the detail and title to the notifier', () => {
    notifyResultError({ errorCode: 'not_authorized', errorDetail: 'raw' }, 'Title')
    expect(notifierError).toHaveBeenCalledWith(notAuthorised(), { title: 'Title', detail: 'raw' })
  })
})
